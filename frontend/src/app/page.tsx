'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '@/context/AuthContext';
import { useRouter } from 'next/navigation';
import ThemeToggle from '@/components/ThemeToggle';
import TaskCard from '@/components/TaskCard';
import TaskDialog from '@/components/TaskDialog';
import { 
  Plus, Search, LogOut, Loader2, Filter, 
  ArrowUpDown, ChevronLeft, ChevronRight, CheckSquare, 
  ListTodo, User, ShieldCheck, AlertCircle 
} from 'lucide-react';

export interface Task {
  id: number;
  user_id: number;
  user_email?: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  due_date: string | null;
  created_at: string;
  updated_at: string;
}

interface PaginationMeta {
  total_items: number;
  total_pages: number;
  current_page: number;
  limit: number;
}

export default function DashboardPage() {
  const { user, logout, apiFetch, isLoading: authLoading } = useAuth();
  const router = useRouter();

  // Task listing states
  const [tasks, setTasks] = useState<Task[]>([]);
  const [pagination, setPagination] = useState<PaginationMeta>({
    total_items: 0,
    total_pages: 1,
    current_page: 1,
    limit: 10,
  });

  // Query parameter states
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [status, setStatus] = useState('');
  const [sortBy, setSortBy] = useState('created_at');
  const [order, setOrder] = useState('desc');
  const [page, setPage] = useState(1);

  // UI Flow states
  const [isTasksLoading, setIsTasksLoading] = useState(true);
  const [tasksError, setTasksError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Dialog state
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);

  // Protect client routing
  useEffect(() => {
    if (!authLoading && !user) {
      router.push('/login');
    }
  }, [user, authLoading, router]);

  // Debounce search input to avoid hitting database on every keystroke
  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1); // Reset page on search
    }, 400);

    return () => clearTimeout(handler);
  }, [search]);

  // Fetch tasks from API
  const fetchTasks = useCallback(async () => {
    if (!user) return;
    setIsTasksLoading(true);
    setTasksError(null);
    try {
      const queryParams = new URLSearchParams({
        status,
        search: debouncedSearch,
        sort_by: sortBy,
        order,
        page: page.toString(),
        limit: '6', // Fetch 6 tasks per page for optimal grid fit
      });

      const response = await apiFetch(`/tasks?${queryParams.toString()}`);
      if (response) {
        setTasks(response.data || []);
        setPagination(response.pagination);
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : 'Failed to fetch tasks.';
      setTasksError(errMsg);
    } finally {
      setIsTasksLoading(false);
    }
  }, [user, status, debouncedSearch, sortBy, order, page, apiFetch]);

  // Trigger tasks fetch on query filter updates
  useEffect(() => {
    setTimeout(() => {
      fetchTasks();
    }, 0);
  }, [fetchTasks]);

  // Action: Add / Update task
  const handleSaveTask = async (taskData: {
    title: string;
    description: string;
    status: string;
    priority: string;
    due_date: string | null;
  }) => {
    try {
      if (editingTask) {
        // Edit Mode
        const updated = await apiFetch(`/tasks/${editingTask.id}`, {
          method: 'PATCH',
          body: JSON.stringify(taskData),
        });
        setTasks(tasks.map((t) => (t.id === editingTask.id ? updated : t)));
        showFeedback('Task updated successfully.');
      } else {
        // Create Mode
        const created = await apiFetch('/tasks', {
          method: 'POST',
          body: JSON.stringify(taskData),
        });
        setTasks([created, ...tasks]);
        showFeedback('Task created successfully.');
      }
      setIsDialogOpen(false);
      setEditingTask(null);
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : 'Failed to save task.';
      throw new Error(errMsg);
    }
  };

  // Action: Open Edit modal
  const handleEditOpen = (task: Task) => {
    setEditingTask(task);
    setIsDialogOpen(true);
  };

  // Action: Open Create modal
  const handleCreateOpen = () => {
    setEditingTask(null);
    setIsDialogOpen(true);
  };

  // Optimistic Action: Delete Task
  const handleDeleteTask = async (taskId: number) => {
    const originalTasks = [...tasks];
    
    // Optimistic UI state change
    setTasks(tasks.filter((t) => t.id !== taskId));
    
    try {
      await apiFetch(`/tasks/${taskId}`, {
        method: 'DELETE',
      });
      showFeedback('Task deleted successfully.');
    } catch (err: unknown) {
      // Rollback on failure
      setTasks(originalTasks);
      const errMsg = err instanceof Error ? err.message : 'Failed to delete task. Restored original list.';
      alert(errMsg);
    }
  };

  // Optimistic Action: Complete Toggle
  const handleToggleComplete = async (task: Task) => {
    const originalTasks = [...tasks];
    const newStatus = task.status === 'completed' ? 'pending' : 'completed';
    
    // Optimistic UI status toggle
    setTasks(
      tasks.map((t) => (t.id === task.id ? { ...t, status: newStatus } : t))
    );

    try {
      await apiFetch(`/tasks/${task.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ status: newStatus }),
      });
    } catch (err: unknown) {
      // Rollback on failure
      setTasks(originalTasks);
      const errMsg = err instanceof Error ? err.message : 'Failed to update task. Rolled back.';
      alert(errMsg);
    }
  };

  const showFeedback = (msg: string) => {
    setSuccessMessage(msg);
    setTimeout(() => setSuccessMessage(null), 3000);
  };

  if (authLoading || !user) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-950">
        <Loader2 className="h-10 w-10 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col pb-12 bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-200 transition-colors duration-200">
      {/* Navigation Header */}
      <header className="sticky top-0 z-40 bg-white/70 dark:bg-slate-900/70 backdrop-blur-md border-b border-slate-200 dark:border-slate-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="h-9 w-9 rounded-xl bg-indigo-600 flex items-center justify-center text-white font-bold shadow-md shadow-indigo-500/20">
              <ListTodo className="h-5 w-5" />
            </div>
            <span className="text-lg font-bold tracking-tight font-sans">
              AetherTasks
            </span>
          </div>

          <div className="flex items-center gap-4">
            {/* User Profile Info */}
            <div className="hidden sm:flex items-center gap-2 border border-slate-200 dark:border-slate-800 rounded-xl px-3.5 py-1.5 bg-slate-50 dark:bg-slate-950/40 text-xs">
              <span className="text-slate-500 flex items-center gap-1">
                {user.role === 'admin' ? (
                  <ShieldCheck className="h-3.5 w-3.5 text-indigo-500 dark:text-indigo-400" />
                ) : (
                  <User className="h-3.5 w-3.5" />
                )}
                {user.email}
              </span>
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
              <span className="font-semibold text-slate-400 uppercase tracking-wide">
                {user.role}
              </span>
            </div>

            <ThemeToggle />

            <button
              onClick={logout}
              className="p-2.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-white/60 dark:bg-slate-900/60 text-slate-500 dark:text-slate-400 hover:text-rose-500 dark:hover:text-rose-400 hover:scale-105 active:scale-95 transition-all duration-200"
              title="Logout"
            >
              <LogOut className="h-5 w-5" />
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 pt-8">
        {/* Dynamic Action Alerts */}
        {successMessage && (
          <div className="fixed bottom-6 right-6 z-50 p-4 rounded-2xl bg-indigo-600 text-white font-semibold text-sm shadow-xl flex items-center gap-2 animate-bounce-slow">
            <CheckSquare className="h-5 w-5" />
            {successMessage}
          </div>
        )}

        {/* Dashboard Title Panel */}
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-8">
          <div>
            <h1 className="text-2xl font-black tracking-tight sm:text-3xl font-sans">
              Task Dashboard
            </h1>
            <p className="mt-1.5 text-sm text-slate-500 dark:text-slate-400">
              {user.role === 'admin' 
                ? 'Managing all tasks registered in the system.' 
                : 'Manage your tasks, deadlines, and project notes.'}
            </p>
          </div>

          <button
            onClick={handleCreateOpen}
            className="w-full md:w-auto inline-flex items-center justify-center px-5 py-3 rounded-2xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold shadow-lg shadow-indigo-500/20 hover:scale-102 active:scale-98 transition duration-150 gap-2 focus:ring-4 focus:ring-indigo-500/20"
          >
            <Plus className="h-5 w-5" />
            New Task
          </button>
        </div>

        {/* Filters Panel */}
        <div className="bg-white dark:bg-slate-900/40 border border-slate-200 dark:border-slate-800 rounded-3xl p-5 mb-8 shadow-xs backdrop-blur-md">
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
            
            {/* Search Input */}
            <div className="relative flex-1 max-w-md">
              <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 h-5 w-5 text-slate-400 dark:text-slate-500" />
              <input
                type="text"
                placeholder="Search tasks by title..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full pl-11 pr-4 py-2.5 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-xl text-sm placeholder-slate-400 text-slate-950 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 transition"
              />
            </div>

            {/* Filter controls */}
            <div className="flex flex-wrap items-center gap-3">
              {/* Status Select */}
              <div className="flex items-center gap-2">
                <Filter className="h-4 w-4 text-slate-400 dark:text-slate-500" />
                <select
                  value={status}
                  onChange={(e) => {
                    setStatus(e.target.value);
                    setPage(1);
                  }}
                  className="px-3.5 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-xl text-xs text-slate-700 dark:text-slate-300 font-semibold focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
                >
                  <option value="">All Statuses</option>
                  <option value="pending">Pending</option>
                  <option value="in_progress">In Progress</option>
                  <option value="completed">Completed</option>
                </select>
              </div>

              {/* Sort field Select */}
              <div className="flex items-center gap-2">
                <ArrowUpDown className="h-4 w-4 text-slate-400 dark:text-slate-500" />
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value)}
                  className="px-3.5 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-xl text-xs text-slate-700 dark:text-slate-300 font-semibold focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
                >
                  <option value="created_at">Created Date</option>
                  <option value="due_date">Due Date</option>
                  <option value="priority">Priority</option>
                </select>
              </div>

              {/* Order Select */}
              <select
                value={order}
                onChange={(e) => setOrder(e.target.value)}
                className="px-3.5 py-2 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-xl text-xs text-slate-700 dark:text-slate-300 font-semibold focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
              >
                <option value="desc">Descending</option>
                <option value="asc">Ascending</option>
              </select>
            </div>

          </div>
        </div>

        {/* Task Grid State Controllers */}
        {tasksError && (
          <div className="mb-6 p-4 rounded-2xl bg-rose-500/10 border border-rose-500/20 text-rose-500 text-sm flex items-center gap-3">
            <AlertCircle className="h-5 w-5" />
            <span>{tasksError}</span>
            <button 
              onClick={fetchTasks}
              className="ml-auto text-xs font-bold underline uppercase tracking-wider text-rose-500 hover:text-rose-600"
            >
              Retry
            </button>
          </div>
        )}

        {isTasksLoading ? (
          /* Loading State Grid Skeleton */
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {[1, 2, 3, 4, 5, 6].map((idx) => (
              <div key={idx} className="border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/40 rounded-2xl p-5 space-y-4 animate-pulse">
                <div className="flex gap-2.5">
                  <div className="h-4 w-12 bg-slate-200 dark:bg-slate-800 rounded" />
                  <div className="h-4 w-16 bg-slate-200 dark:bg-slate-800 rounded" />
                </div>
                <div className="h-6 w-3/4 bg-slate-200 dark:bg-slate-800 rounded" />
                <div className="space-y-2">
                  <div className="h-3.5 w-full bg-slate-200 dark:bg-slate-800 rounded" />
                  <div className="h-3.5 w-5/6 bg-slate-200 dark:bg-slate-800 rounded" />
                </div>
                <div className="flex items-center gap-2 pt-2">
                  <div className="h-3 w-4 bg-slate-200 dark:bg-slate-800 rounded" />
                  <div className="h-3 w-24 bg-slate-200 dark:bg-slate-800 rounded" />
                </div>
              </div>
            ))}
          </div>
        ) : tasks.length === 0 ? (
          /* Empty State Illustration */
          <div className="border border-slate-200 dark:border-slate-800 rounded-3xl bg-white dark:bg-slate-900/30 backdrop-blur-xs py-16 px-4 text-center">
            <div className="inline-flex p-4 rounded-full bg-indigo-500/5 dark:bg-indigo-500/10 border border-indigo-500/10 text-indigo-400 mb-4">
              <ListTodo className="h-10 w-10 text-slate-400 dark:text-slate-600" />
            </div>
            <h3 className="text-lg font-bold text-slate-900 dark:text-white">
              No tasks found
            </h3>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400 max-w-sm mx-auto">
              We couldn&apos;t find any tasks matching your filters. Try updating your filters or create a new task to get started!
            </p>
          </div>
        ) : (
          /* Task Grid List */
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {tasks.map((task) => (
              <TaskCard
                key={task.id}
                task={task}
                onEdit={handleEditOpen}
                onDelete={handleDeleteTask}
                onToggleComplete={handleToggleComplete}
              />
            ))}
          </div>
        )}

        {/* Pagination Controllers */}
        {pagination.total_pages > 1 && (
          <div className="mt-10 flex items-center justify-between border-t border-slate-200 dark:border-slate-800 pt-6">
            <p className="text-xs text-slate-400 dark:text-slate-500">
              Showing page <span className="font-semibold text-slate-700 dark:text-slate-300">{pagination.current_page}</span> of{' '}
              <span className="font-semibold text-slate-700 dark:text-slate-300">{pagination.total_pages}</span> (
              <span className="font-semibold text-slate-700 dark:text-slate-300">{pagination.total_items}</span> items)
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
                className="inline-flex items-center gap-1 px-3.5 py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 text-xs font-semibold hover:bg-slate-50 dark:hover:bg-slate-950 transition active:scale-95 disabled:opacity-50"
              >
                <ChevronLeft className="h-4 w-4" />
                Prev
              </button>
              <button
                onClick={() => setPage(Math.min(pagination.total_pages, page + 1))}
                disabled={page === pagination.total_pages}
                className="inline-flex items-center gap-1 px-3.5 py-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 text-xs font-semibold hover:bg-slate-50 dark:hover:bg-slate-950 transition active:scale-95 disabled:opacity-50"
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        )}
      </main>

      {/* Task Modal Dialog Form */}
      <TaskDialog
        isOpen={isDialogOpen}
        onClose={() => {
          setIsDialogOpen(false);
          setEditingTask(null);
        }}
        onSave={handleSaveTask}
        task={editingTask}
      />
    </div>
  );
}
