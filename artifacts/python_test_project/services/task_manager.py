"""
Task management service for handling task operations.
"""

from typing import List, Optional, Callable, Any
from datetime import datetime, timedelta

from models.task import Task, Priority, Status
from models.project import Project


class TaskManager:
    """
    Service class for managing tasks within a project.
    
    Provides high-level operations for task management including
    filtering, sorting, and bulk operations.
    """
    
    def __init__(self, project: Project):
        """Initialize task manager with a project."""
        self.project = project
    
    def add_task(self, task: Task) -> None:
        """Add a task to the project."""
        self.project.add_task(task)
    
    def remove_task(self, task_id: str) -> bool:
        """Remove a task from the project."""
        return self.project.remove_task(task_id)
    
    def get_all_tasks(self) -> List[Task]:
        """Get all tasks in the project."""
        return self.project.tasks.copy()
    
    def find_tasks_by_tag(self, tag: str) -> List[Task]:
        """Find all tasks with a specific tag."""
        return [
            task for task in self.project.tasks
            if tag.lower() in task.tags
        ]
    
    def find_tasks_by_assignee(self, assignee: str) -> List[Task]:
        """Find all tasks assigned to a specific person."""
        return [
            task for task in self.project.tasks
            if task.assignee and task.assignee.lower() == assignee.lower()
        ]
    
    def find_tasks_by_priority(self, priority: Priority) -> List[Task]:
        """Find all tasks with a specific priority."""
        return self.project.get_tasks_by_priority(priority)
    
    def find_tasks_due_within(self, days: int) -> List[Task]:
        """Find tasks due within the specified number of days."""
        cutoff_date = datetime.now() + timedelta(days=days)
        return [
            task for task in self.project.tasks
            if task.due_date and task.due_date <= cutoff_date
            and task.status != Status.DONE
        ]
    
    def search_tasks(self, query: str) -> List[Task]:
        """Search tasks by title or description."""
        query_lower = query.lower()
        return [
            task for task in self.project.tasks
            if query_lower in task.title.lower()
            or query_lower in task.description.lower()
        ]
    
    def sort_tasks(self, key: Callable[[Task], Any], reverse: bool = False) -> List[Task]:
        """Sort tasks by a given key function."""
        return sorted(self.project.tasks, key=key, reverse=reverse)
    
    def sort_by_priority(self) -> List[Task]:
        """Sort tasks by priority (highest first)."""
        priority_order = {
            Priority.CRITICAL: 0,
            Priority.HIGH: 1,
            Priority.MEDIUM: 2,
            Priority.LOW: 3
        }
        return self.sort_tasks(lambda t: priority_order[t.priority])
    
    def sort_by_due_date(self) -> List[Task]:
        """Sort tasks by due date (earliest first)."""
        # Tasks without due dates go to the end
        def due_date_key(task: Task) -> datetime:
            return task.due_date or datetime.max
        
        return self.sort_tasks(due_date_key)
    
    def bulk_update_status(self, task_ids: List[str], new_status: Status) -> int:
        """Update status for multiple tasks."""
        updated_count = 0
        
        for task_id in task_ids:
            task = self.project.get_task(task_id)
            if task and task.status != new_status:
                if new_status == Status.DONE:
                    task.complete()
                elif new_status == Status.IN_PROGRESS:
                    task.start()
                elif new_status == Status.CANCELLED:
                    task.cancel()
                else:
                    task.status = new_status
                updated_count += 1
        
        if updated_count > 0:
            self.project.updated_at = datetime.now()
        
        return updated_count
    
    def get_productivity_score(self) -> float:
        """
        Calculate a productivity score based on task completion rate
        and time to completion.
        """
        if self.project.total_tasks == 0:
            return 0.0
        
        completed_tasks = [t for t in self.project.tasks if t.is_completed]
        if not completed_tasks:
            return 0.0
        
        # Base score from completion percentage
        base_score = self.project.completion_percentage
        
        # Bonus for completing tasks before due date
        on_time_bonus = 0.0
        for task in completed_tasks:
            if task.due_date and task.completed_at:
                if task.completed_at <= task.due_date:
                    on_time_bonus += 10.0
        
        # Normalize bonus
        on_time_bonus = min(on_time_bonus / len(completed_tasks), 20.0)
        
        return min(base_score + on_time_bonus, 100.0)