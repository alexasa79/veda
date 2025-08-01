"""
Project model for grouping related tasks.
"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Optional, Dict, Any
from uuid import uuid4

from .task import Task, Status, Priority


@dataclass
class Project:
    """
    Represents a project containing multiple tasks.
    
    A project groups related tasks and provides
    aggregate statistics and management capabilities.
    """
    
    name: str
    description: str
    created_at: datetime = field(default_factory=datetime.now)
    updated_at: datetime = field(default_factory=datetime.now)
    owner: Optional[str] = None
    id: str = field(default_factory=lambda: str(uuid4()))
    tasks: List[Task] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    def add_task(self, task: Task) -> None:
        """Add a task to the project."""
        self.tasks.append(task)
        self.updated_at = datetime.now()
    
    def remove_task(self, task_id: str) -> bool:
        """Remove a task by ID."""
        initial_count = len(self.tasks)
        self.tasks = [t for t in self.tasks if t.id != task_id]
        
        if len(self.tasks) < initial_count:
            self.updated_at = datetime.now()
            return True
        return False
    
    def get_task(self, task_id: str) -> Optional[Task]:
        """Get a task by ID."""
        for task in self.tasks:
            if task.id == task_id:
                return task
        return None
    
    def get_tasks_by_status(self, status: Status) -> List[Task]:
        """Get all tasks with a specific status."""
        return [task for task in self.tasks if task.status == status]
    
    def get_tasks_by_priority(self, priority: Priority) -> List[Task]:
        """Get all tasks with a specific priority."""
        return [task for task in self.tasks if task.priority == priority]
    
    def get_overdue_tasks(self) -> List[Task]:
        """Get all overdue tasks."""
        return [task for task in self.tasks if task.is_overdue()]
    
    @property
    def total_tasks(self) -> int:
        """Total number of tasks in the project."""
        return len(self.tasks)
    
    @property
    def completed_tasks(self) -> int:
        """Number of completed tasks."""
        return len([t for t in self.tasks if t.is_completed])
    
    @property
    def completion_percentage(self) -> float:
        """Calculate project completion percentage."""
        if self.total_tasks == 0:
            return 0.0
        return (self.completed_tasks / self.total_tasks) * 100
    
    def get_statistics(self) -> Dict[str, Any]:
        """Get project statistics."""
        stats = {
            'total_tasks': self.total_tasks,
            'completed_tasks': self.completed_tasks,
            'completion_percentage': self.completion_percentage,
            'overdue_tasks': len(self.get_overdue_tasks()),
            'tasks_by_status': {},
            'tasks_by_priority': {}
        }
        
        # Count tasks by status
        for status in Status:
            count = len(self.get_tasks_by_status(status))
            if count > 0:
                stats['tasks_by_status'][status.value] = count
        
        # Count tasks by priority
        for priority in Priority:
            count = len(self.get_tasks_by_priority(priority))
            if count > 0:
                stats['tasks_by_priority'][str(priority)] = count
        
        return stats
    
    def __repr__(self) -> str:
        return f"Project(name='{self.name}', tasks={self.total_tasks})"