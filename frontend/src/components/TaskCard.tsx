'use client';

import React, { useState } from 'react';
import { Calendar, Trash2, Edit3, ClipboardList, CheckCircle, Circle, ChevronDown, ChevronUp, Clock, User, Loader2 } from 'lucide-react';
import { Task } from '@/app/page';
import { useAuth } from '@/context/AuthContext';

interface TaskCardProps {
  task: Task;
  onEdit: (task: Task) => void;
  onDelete: (id: number) => Promise<void>;
  onToggleComplete: (task: Task) => Promise<void>;
}

interface ActivityLog {
  id: number;
  task_id: number;
  user_id: number;
  user_email: string;
  action: string;
  details: string;
  created_at: string;
}

export default function TaskCard({ task, onEdit, onDelete, onToggleComplete }: TaskCardProps) {
  const { apiFetch, user } = useAuth();
  const [showLogs, setShowLogs] = useState(false);
  const [logs, setLogs] = useState<ActivityLog[]>([]);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isToggling, setIsToggling] = useState(false);

  const isAdmin = user?.role === 'admin';

  // Format Date for readability
  const formatDueDate = (dateStr: string | null) => {
    if (!dateStr) return null;
    const date = new Date(dateStr);
    return date.toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // Toggle log panel visibility & load audit records from backend
  const handleToggleLogs = async () => {
    if (!showLogs && logs.length === 0) {
      setIsLoadingLogs(true);
      try {
        const data = await apiFetch(`/tasks/${task.id}/logs`);
        setLogs(data || []);
      } catch (err) {
        console.error('Failed to load logs', err);
      } finally {
        setIsLoadingLogs(false);
      }
    }
    setShowLogs(!showLogs);
  };

  const handleToggleCheck = async () => {
    setIsToggling(true);
    try {
      await onToggleComplete(task);
    } finally {
      setIsToggling(false);
    }
  };

  const handleDeleteClick = async () => {
    if (confirm('Are you sure you want to delete this task?')) {
      setIsDeleting(true);
      try {
        await onDelete(task.id);
      } catch {
        setIsDeleting(false);
      }
    }
  };

  // Status-based formatting helpers
  const isCompleted = task.status === 'completed';
  const isInProgress = task.status === 'in_progress';

  // Priority color mappings
  const priorityStyles = {
    high: 'bg-rose-500/10 border-rose-500/20 text-rose-500 dark:text-rose-400',
    medium: 'bg-amber-500/10 border-amber-500/20 text-amber-500 dark:text-amber-400',
    low: 'bg-emerald-500/10 border-emerald-500/20 text-emerald-500 dark:text-emerald-400',
  }[task.priority as 'high' | 'medium' | 'low'] || 'bg-slate-500/10 border-slate-500/20 text-slate-500';

  const parseLogDetails = (detailsStr: string) => {
    try {
      const details = JSON.parse(detailsStr);
      return Object.keys(details).map((key) => {
        const val = details[key];
        if (typeof val === 'object' && val !== null && 'from' in val && 'to' in val) {
          const from = val.from === null || val.from === '' ? 'None' : String(val.from);
          const to = val.to === null || val.to === '' ? 'None' : String(val.to);
          return `${key}: ${from} → ${to}`;
        }
        return `${key}: ${String(val)}`;
      }).join(', ');
    } catch {
      return detailsStr;
    }
  };

  return (
    <div 
      className={`border rounded-2xl bg-white dark:bg-slate-900/60 backdrop-blur-xs shadow-md transition-all duration-300 ${
        isCompleted 
          ? 'border-slate-200 dark:border-slate-800 opacity-70' 
          : isInProgress
          ? 'border-indigo-200 dark:border-indigo-900/40 shadow-indigo-500/5'
          : 'border-slate-200 dark:border-slate-800'
      }`}
    >
      <div className="p-5 flex gap-4">
        {/* Checkbox button */}
        <button
          onClick={handleToggleCheck}
          disabled={isToggling}
          className="mt-0.5 text-slate-400 hover:text-indigo-600 dark:text-slate-600 dark:hover:text-indigo-400 transition focus:outline-none h-fit"
        >
          {isCompleted ? (
            <CheckCircle className="h-6 w-6 text-indigo-500 animate-pulse-once" />
          ) : (
            <Circle className="h-6 w-6 text-slate-300 dark:text-slate-700" />
          )}
        </button>

        {/* Task Details */}
        <div className="flex-1 min-w-0">
          <div className="flex flex-wrap items-center gap-2 mb-1.5">
            {/* Priority Badge */}
            <span className={`text-[10px] font-bold tracking-wider uppercase px-2 py-0.5 rounded-md border ${priorityStyles}`}>
              {task.priority}
            </span>

            {/* Status Badge */}
            <span className={`text-[10px] font-bold tracking-wider uppercase px-2 py-0.5 rounded-md border ${
              isCompleted 
                ? 'bg-slate-100 dark:bg-slate-800 border-slate-200 dark:border-slate-700 text-slate-500' 
                : isInProgress 
                ? 'bg-indigo-500/10 border-indigo-500/20 text-indigo-500 dark:text-indigo-400' 
                : 'bg-slate-100 dark:bg-slate-800 border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400'
            }`}>
              {task.status.replace('_', ' ')}
            </span>

            {/* Admin specific owner information */}
            {isAdmin && task.user_email && (
              <span className="text-[10px] flex items-center gap-1 text-indigo-500 dark:text-indigo-400 bg-indigo-500/5 dark:bg-indigo-500/10 px-2 py-0.5 rounded-md font-medium">
                <User className="h-2.5 w-2.5" />
                {task.user_email}
              </span>
            )}
          </div>

          <h3 className={`text-base font-bold text-slate-900 dark:text-white truncate ${isCompleted ? 'line-through text-slate-400 dark:text-slate-500' : ''}`}>
            {task.title}
          </h3>

          {task.description && (
            <p className={`mt-1 text-sm text-slate-500 dark:text-slate-400 line-clamp-2 ${isCompleted ? 'text-slate-400/80 dark:text-slate-500/80' : ''}`}>
              {task.description}
            </p>
          )}

          {/* Metadata: Due Date & Created Date */}
          <div className="mt-4 flex flex-wrap gap-x-4 gap-y-2 text-xs text-slate-400 dark:text-slate-500">
            {task.due_date && (
              <span className="flex items-center gap-1.5 font-medium text-slate-500 dark:text-slate-400">
                <Calendar className="h-3.5 w-3.5" />
                Due: {formatDueDate(task.due_date)}
              </span>
            )}
            <span className="flex items-center gap-1.5">
              <Clock className="h-3.5 w-3.5" />
              Created: {new Date(task.created_at).toLocaleDateString()}
            </span>
          </div>
        </div>

        {/* Card Operations */}
        <div className="flex flex-col gap-2.5 justify-start">
          <div className="flex items-center gap-1">
            <button
              onClick={() => onEdit(task)}
              className="p-1.5 rounded-lg text-slate-400 hover:text-indigo-600 dark:text-slate-500 dark:hover:text-indigo-400 hover:bg-slate-50 dark:hover:bg-slate-800 transition"
              title="Edit Task"
            >
              <Edit3 className="h-4.5 w-4.5" />
            </button>
            <button
              onClick={handleDeleteClick}
              disabled={isDeleting}
              className="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 dark:text-slate-500 dark:hover:text-rose-400 hover:bg-slate-50 dark:hover:bg-slate-800 transition disabled:opacity-50"
              title="Delete Task"
            >
              <Trash2 className="h-4.5 w-4.5" />
            </button>
          </div>

          <button
            onClick={handleToggleLogs}
            className="flex items-center gap-1 text-[11px] font-semibold text-slate-400 dark:text-slate-500 hover:text-indigo-500 dark:hover:text-indigo-400 border border-transparent hover:border-slate-100 dark:hover:border-slate-800 px-2 py-1 rounded-lg transition"
          >
            <ClipboardList className="h-3.5 w-3.5" />
            History
            {showLogs ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
          </button>
        </div>
      </div>

      {/* Expandable Activity Logs Drawer */}
      {showLogs && (
        <div className="border-t border-slate-100 dark:border-slate-800/80 bg-slate-50/50 dark:bg-slate-950/20 px-5 py-4 rounded-b-2xl">
          <h4 className="text-xs font-bold uppercase tracking-wider text-slate-400 dark:text-slate-500 mb-3 flex items-center gap-1.5">
            <ClipboardList className="h-3.5 w-3.5" />
            Task Activity Log
          </h4>

          {isLoadingLogs ? (
            <div className="flex items-center gap-2 text-xs text-slate-400 dark:text-slate-500 py-2">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Loading history trail...
            </div>
          ) : logs.length === 0 ? (
            <div className="text-xs text-slate-400 dark:text-slate-500 py-1">
              No modifications recorded.
            </div>
          ) : (
            <div className="space-y-3 relative before:absolute before:left-2 before:top-2 before:bottom-2 before:w-0.5 before:bg-slate-200 dark:before:bg-slate-800">
              {logs.map((log) => (
                <div key={log.id} className="text-xs relative pl-6">
                  {/* Timeline dot */}
                  <div className="absolute left-1 top-1.5 w-2 h-2 rounded-full bg-slate-300 dark:bg-slate-700 -translate-x-0.5" />
                  
                  <div className="flex items-center justify-between gap-4">
                    <span className="font-semibold text-slate-700 dark:text-slate-300">
                      {log.action.replace('_', ' ')}
                    </span>
                    <span className="text-[10px] text-slate-400 dark:text-slate-500">
                      {new Date(log.created_at).toLocaleString()}
                    </span>
                  </div>
                  
                  <div className="text-slate-500 dark:text-slate-400 mt-0.5 leading-relaxed">
                    {log.action === 'created' ? 'Task created.' : parseLogDetails(log.details)}
                  </div>
                  
                  <div className="text-[10px] text-slate-400 dark:text-slate-500 mt-0.5 italic flex items-center gap-1">
                    <User className="h-2.5 w-2.5" />
                    By: {log.user_email}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
