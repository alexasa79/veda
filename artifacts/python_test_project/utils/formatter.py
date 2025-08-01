"""
Formatters for displaying tasks and projects.
"""

import json
from abc import ABC, abstractmethod
from typing import Dict, Any, List
from datetime import datetime

from models.task import Task, Priority, Status
from models.project import Project


class Formatter(ABC):
    """Abstract base class for formatters."""
    
    @abstractmethod
    def format_task(self, task: Task) -> str:
        """Format a single task."""
        pass
    
    @abstractmethod
    def format_project(self, project: Project) -> str:
        """Format a project."""
        pass


class ConsoleFormatter(Formatter):
    """Formatter for console output with colors and styling."""
    
    # ANSI color codes
    COLORS = {
        'red': '\033[91m',
        'green': '\033[92m',
        'yellow': '\033[93m',
        'blue': '\033[94m',
        'magenta': '\033[95m',
        'cyan': '\033[96m',
        'white': '\033[97m',
        'reset': '\033[0m',
        'bold': '\033[1m',
        'underline': '\033[4m'
    }
    
    def __init__(self, use_colors: bool = True):
        """Initialize formatter with optional color support."""
        self.use_colors = use_colors
    
    def _colorize(self, text: str, color: str) -> str:
        """Apply color to text if colors are enabled."""
        if not self.use_colors or color not in self.COLORS:
            return text
        return f"{self.COLORS[color]}{text}{self.COLORS['reset']}"
    
    def _format_priority(self, priority: Priority) -> str:
        """Format priority with color."""
        colors = {
            Priority.LOW: 'blue',
            Priority.MEDIUM: 'yellow',
            Priority.HIGH: 'red',
            Priority.CRITICAL: 'magenta'
        }
        color = colors.get(priority, 'white')
        return self._colorize(str(priority), color)
    
    def _format_status(self, status: Status) -> str:
        """Format status with color."""
        colors = {
            Status.TODO: 'white',
            Status.IN_PROGRESS: 'yellow',
            Status.DONE: 'green',
            Status.CANCELLED: 'red'
        }
        color = colors.get(status, 'white')
        return self._colorize(status.value.upper(), color)
    
    def _format_date(self, date: datetime) -> str:
        """Format datetime as string."""
        return date.strftime("%Y-%m-%d %H:%M")
    
    def format_task(self, task: Task) -> str:
        """Format a task for console display."""
        lines = []
        
        # Title with ID
        title_line = f"{self._colorize('Task:', 'bold')} {task.title}"
        if task.is_completed:
            title_line = self._colorize(f"✓ {title_line}", 'green')
        lines.append(title_line)
        
        # ID
        lines.append(f"  {self._colorize('ID:', 'cyan')} {task.id[:8]}")
        
        # Status and Priority
        lines.append(
            f"  {self._colorize('Status:', 'cyan')} {self._format_status(task.status)} | "
            f"{self._colorize('Priority:', 'cyan')} {self._format_priority(task.priority)}"
        )
        
        # Description
        if task.description:
            lines.append(f"  {self._colorize('Description:', 'cyan')} {task.description}")
        
        # Dates
        if task.due_date:
            due_str = self._format_date(task.due_date)
            if task.is_overdue():
                due_str = self._colorize(f"{due_str} (OVERDUE)", 'red')
            lines.append(f"  {self._colorize('Due:', 'cyan')} {due_str}")
        
        # Tags
        if task.tags:
            tags_str = ', '.join(f"#{tag}" for tag in sorted(task.tags))
            lines.append(f"  {self._colorize('Tags:', 'cyan')} {tags_str}")
        
        # Assignee
        if task.assignee:
            lines.append(f"  {self._colorize('Assignee:', 'cyan')} {task.assignee}")
        
        return '\n'.join(lines)
    
    def format_project(self, project: Project) -> str:
        """Format a project for console display."""
        lines = []
        
        # Header
        lines.append(self._colorize(f"Project: {project.name}", 'bold'))
        lines.append(self._colorize("=" * (len(project.name) + 9), 'bold'))
        
        # Description
        if project.description:
            lines.append(f"{project.description}")
            lines.append("")
        
        # Statistics
        stats = project.get_statistics()
        lines.append(self._colorize("Statistics:", 'underline'))
        lines.append(f"  Total Tasks: {stats['total_tasks']}")
        lines.append(f"  Completed: {stats['completed_tasks']} ({stats['completion_percentage']:.1f}%)")
        
        if stats['overdue_tasks'] > 0:
            lines.append(self._colorize(f"  Overdue: {stats['overdue_tasks']}", 'red'))
        
        # Tasks by status
        if stats['tasks_by_status']:
            lines.append("")
            lines.append(self._colorize("Tasks by Status:", 'underline'))
            for status, count in stats['tasks_by_status'].items():
                lines.append(f"  {status}: {count}")
        
        # Tasks by priority
        if stats['tasks_by_priority']:
            lines.append("")
            lines.append(self._colorize("Tasks by Priority:", 'underline'))
            for priority, count in stats['tasks_by_priority'].items():
                lines.append(f"  {priority}: {count}")
        
        return '\n'.join(lines)


class JSONFormatter(Formatter):
    """Formatter for JSON output."""
    
    def __init__(self, indent: int = 2):
        """Initialize JSON formatter with indentation level."""
        self.indent = indent
    
    def _task_to_dict(self, task: Task) -> Dict[str, Any]:
        """Convert task to dictionary."""
        return {
            'id': task.id,
            'title': task.title,
            'description': task.description,
            'priority': str(task.priority),
            'status': task.status.value,
            'created_at': task.created_at.isoformat(),
            'due_date': task.due_date.isoformat() if task.due_date else None,
            'completed_at': task.completed_at.isoformat() if task.completed_at else None,
            'tags': list(task.tags),
            'assignee': task.assignee,
            'is_overdue': task.is_overdue(),
            'is_completed': task.is_completed
        }
    
    def format_task(self, task: Task) -> str:
        """Format a task as JSON."""
        return json.dumps(self._task_to_dict(task), indent=self.indent)
    
    def format_project(self, project: Project) -> str:
        """Format a project as JSON."""
        data = {
            'id': project.id,
            'name': project.name,
            'description': project.description,
            'created_at': project.created_at.isoformat(),
            'updated_at': project.updated_at.isoformat(),
            'owner': project.owner,
            'statistics': project.get_statistics(),
            'tasks': [self._task_to_dict(task) for task in project.tasks]
        }
        return json.dumps(data, indent=self.indent)