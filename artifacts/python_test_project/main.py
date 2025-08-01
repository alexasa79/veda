#!/usr/bin/env python3
"""
Main entry point for the task management system.
"""

import sys
from typing import List
from datetime import datetime

from models.task import Task, Priority
from models.project import Project
from services.task_manager import TaskManager
from utils.formatter import ConsoleFormatter
from utils.validators import validate_date_string


def create_sample_tasks() -> List[Task]:
    """Create some sample tasks for demonstration."""
    tasks = [
        Task(
            title="Implement user authentication",
            description="Add login/logout functionality with JWT tokens",
            priority=Priority.HIGH,
            due_date=datetime(2024, 12, 31)
        ),
        Task(
            title="Write unit tests",
            description="Achieve 80% code coverage",
            priority=Priority.MEDIUM
        ),
        Task(
            title="Update documentation",
            description="Add API documentation and usage examples",
            priority=Priority.LOW,
            tags=["docs", "api"]
        )
    ]
    return tasks


def main():
    """Main application entry point."""
    print("=== Task Management System ===\n")
    
    # Create a project
    project = Project("Python Test Project", "A sample project for testing")
    
    # Create task manager
    manager = TaskManager(project)
    
    # Add sample tasks
    sample_tasks = create_sample_tasks()
    for task in sample_tasks:
        manager.add_task(task)
    
    # Create formatter
    formatter = ConsoleFormatter()
    
    # Display project info
    print(formatter.format_project(project))
    print("\n" + "="*50 + "\n")
    
    # Display all tasks
    print("All Tasks:")
    for task in manager.get_all_tasks():
        print(formatter.format_task(task))
        print()
    
    # Find high priority tasks
    high_priority = manager.find_tasks_by_priority(Priority.HIGH)
    print(f"\nHigh Priority Tasks: {len(high_priority)}")
    
    # Complete a task
    if sample_tasks:
        first_task = sample_tasks[0]
        first_task.complete()
        print(f"\nCompleted: {first_task.title}")
        print(f"Completion time: {first_task.completed_at}")
    
    # Test date validation
    test_dates = ["2024-12-25", "invalid-date", "2024-13-45"]
    print("\n\nDate Validation Tests:")
    for date_str in test_dates:
        is_valid = validate_date_string(date_str)
        print(f"  '{date_str}' -> {'Valid' if is_valid else 'Invalid'}")


if __name__ == "__main__":
    main()