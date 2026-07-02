package service

import (
	"context"
	"strings"
)

func (s *TempTaskService) ListTempTasks(ctx context.Context, filter TempTaskFilter) ([]TempTask, error) {
	return s.repo.ListTempTasks(ctx, filter)
}

func (s *TempTaskService) GetTempTask(ctx context.Context, id int64) (TempTask, error) {
	task, err := s.repo.GetTempTask(ctx, id)
	return task, mapError(err)
}

func (s *TempTaskService) CreateTempTask(ctx context.Context, task TempTask) (TempTask, error) {
	now := nowString()
	task.Title = strings.TrimSpace(task.Title)
	defaultTaskFields(&task)
	if err := validateTempTask(task); err != nil {
		return TempTask{}, err
	}
	task.CreatedAt = now
	task.UpdatedAt = now
	var created TempTask
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		created, err = repo.CreateTempTask(ctx, task)
		if err != nil {
			return err
		}
		if err := recordTempTaskActivity(ctx, repo, created, "created", ""); err != nil {
			return err
		}
		return indexTempTask(ctx, repo, created)
	}); err != nil {
		return TempTask{}, err
	}
	return created, nil
}

func (s *TempTaskService) UpdateTempTask(ctx context.Context, id int64, task TempTask) (TempTask, error) {
	existing, err := s.GetTempTask(ctx, id)
	if err != nil {
		return TempTask{}, err
	}
	task.ID = existing.ID
	task.CreatedAt = existing.CreatedAt
	task.UpdatedAt = nowString()
	task.Title = strings.TrimSpace(task.Title)
	defaultTaskFields(&task)
	if err := validateTempTask(task); err != nil {
		return TempTask{}, err
	}
	var updated TempTask
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		updated, err = repo.UpdateTempTask(ctx, task)
		if err != nil {
			return err
		}
		return indexTempTask(ctx, repo, updated)
	}); err != nil {
		return TempTask{}, err
	}
	return updated, nil
}

func (s *TempTaskService) DeleteTempTask(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteTempTask(ctx, id))
}

func (s *TempTaskService) ListTempTaskEvents(ctx context.Context, taskID int64) ([]TempTaskEvent, error) {
	if _, err := s.GetTempTask(ctx, taskID); err != nil {
		return nil, err
	}
	return s.repo.ListTempTaskEvents(ctx, taskID)
}

func (s *TempTaskService) CreateTempTaskEvent(ctx context.Context, taskID int64, event TempTaskEvent) (TempTaskEvent, error) {
	task, err := s.GetTempTask(ctx, taskID)
	if err != nil {
		return TempTaskEvent{}, err
	}
	now := nowString()
	event.TempTaskID = taskID
	event.EventType = strings.TrimSpace(event.EventType)
	event.ContentMD = strings.TrimSpace(event.ContentMD)
	if event.HappenedAt == "" {
		event.HappenedAt = now
	}
	event.CreatedAt = now
	event.UpdatedAt = now
	if err := validateTempTaskEvent(event); err != nil {
		return TempTaskEvent{}, err
	}
	var created TempTaskEvent
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		created, err = repo.CreateTempTaskEvent(ctx, event)
		if err != nil {
			return err
		}
		return indexTempTaskEvent(ctx, repo, task, created)
	}); err != nil {
		return TempTaskEvent{}, err
	}
	return created, nil
}

func (s *TempTaskService) UpdateTempTaskEvent(ctx context.Context, id int64, event TempTaskEvent) (TempTaskEvent, error) {
	existing, err := s.repo.GetTempTaskEvent(ctx, id)
	if err != nil {
		return TempTaskEvent{}, mapError(err)
	}
	event.ID = id
	event.TempTaskID = existing.TempTaskID
	event.CreatedAt = existing.CreatedAt
	event.EventType = strings.TrimSpace(event.EventType)
	event.ContentMD = strings.TrimSpace(event.ContentMD)
	event.UpdatedAt = nowString()
	if event.HappenedAt == "" {
		event.HappenedAt = existing.HappenedAt
	}
	if err := validateTempTaskEvent(event); err != nil {
		return TempTaskEvent{}, err
	}
	var updated TempTaskEvent
	if err := s.withRepository(ctx, func(repo Repository) error {
		var err error
		updated, err = repo.UpdateTempTaskEvent(ctx, event)
		if err != nil {
			return err
		}
		task, err := repo.GetTempTask(ctx, updated.TempTaskID)
		if err != nil {
			return err
		}
		return indexTempTaskEvent(ctx, repo, task, updated)
	}); err != nil {
		return TempTaskEvent{}, err
	}
	return updated, nil
}

func (s *TempTaskService) DeleteTempTaskEvent(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteTempTaskEvent(ctx, id))
}
