package space

import (
	"context"
	"fmt"
	"log/slog"

	domainspace "github.com/whicu/slotify/internal/domain/space"
)

type ListSpaces struct {
	log    *slog.Logger
	spaces SpaceRepository
}

func NewListSpaces(
	log *slog.Logger,
	spaces SpaceRepository,
) *ListSpaces {
	return &ListSpaces{
		log:    log,
		spaces: spaces,
	}
}

type ListSpacesInput struct {
	OnlyActive bool
}

type ListSpacesItem struct {
	ID       domainspace.SpaceID
	Name     string
	Type     string
	Capacity int
	Active   bool
}

type ListSpacesOutput struct {
	Spaces []ListSpacesItem
}

func (l *ListSpaces) Execute(ctx context.Context, in ListSpacesInput) (*ListSpacesOutput, error) {
	l.log.DebugContext(ctx, "executing list spaces", slog.Bool("only_active", in.OnlyActive))

	var (
		all []*domainspace.Space
		err error
	)

	if in.OnlyActive {
		all, err = l.spaces.ListActive(ctx)
	} else {
		all, err = l.spaces.ListAll(ctx)
	}

	if err != nil {
		l.log.ErrorContext(ctx, "failed to list spaces", slog.Any("error", err))
		return nil, fmt.Errorf("list spaces: %w", err)
	}

	items := make([]ListSpacesItem, 0, len(all))
	for _, s := range all {
		items = append(items, ListSpacesItem{
			ID:       s.ID(),
			Name:     s.Name(),
			Type:     s.Type().String(),
			Capacity: s.Capacity(),
			Active:   s.IsActive(),
		})
	}

	l.log.InfoContext(ctx, "spaces listed",
		slog.Int("count", len(items)),
		slog.Bool("only_active", in.OnlyActive),
	)

	return &ListSpacesOutput{Spaces: items}, nil
}
