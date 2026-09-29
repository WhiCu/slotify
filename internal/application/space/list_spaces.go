package space

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	domainspace "github.com/whicu/slotify/internal/domain/space"
)

type ListSpacesLister interface {
	List(
		ctx context.Context,
		onlyActive bool,
	) ([]*domainspace.Space, error)
}

type ListSpaces struct {
	log    *slog.Logger
	spaces ListSpacesLister
}

func NewListSpaces(
	log *slog.Logger,
	spaces ListSpacesLister,
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
	ID        domainspace.SpaceID
	Name      string
	Type      string
	Capacity  int
	Active    bool
	CreatedAt time.Time
}

type ListSpacesOutput struct {
	Spaces []ListSpacesItem
}

func (l *ListSpaces) Execute(
	ctx context.Context,
	in ListSpacesInput,
) (*ListSpacesOutput, error) {
	l.log.DebugContext(
		ctx,
		"executing list spaces",
		slog.Bool("only_active", in.OnlyActive),
	)

	all, err := l.spaces.List(ctx, in.OnlyActive)
	if err != nil {
		return nil, fmt.Errorf("list spaces: %w", err)
	}

	items := make([]ListSpacesItem, 0, len(all))

	for _, s := range all {
		items = append(items, ListSpacesItem{
			ID:        s.ID(),
			Name:      s.Name(),
			Type:      s.Type().String(),
			Capacity:  s.Capacity(),
			Active:    s.IsActive(),
			CreatedAt: s.CreatedAt(),
		})
	}

	l.log.InfoContext(
		ctx,
		"spaces listed",
		slog.Int("count", len(items)),
		slog.Bool("only_active", in.OnlyActive),
	)

	return &ListSpacesOutput{
		Spaces: items,
	}, nil
}
