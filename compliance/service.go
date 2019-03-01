package compliance

import "context"

type Service interface {
	CreateEntry(ctx context.Context, feature *Feature) error
	GetLastIfDiffers(ctx context.Context, feature *Feature) (bool, *Feature, error)
	//Create(ctx context.Context, dog *Dog) error
	//Get(ctx context.Context, id uint64) (*Dog, error)
	//List(ctx context.Context) (Dogs, error)
	//Update(ctx context.Context, dog *Dog) error
	//Delete(ctx context.Context, dog *Dog) error
	Close(ctx context.Context) error
}
