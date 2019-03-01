package compliance

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type SqliteService struct {
	db *sqlx.DB
}

func NewSqliteService(db *sqlx.DB) *SqliteService {
	return &SqliteService{
		db: db,
	}
}

func (s *SqliteService) CreateEntry(ctx context.Context, feature *Feature) error {
	query := `INSERT INTO features
		(name, timestamp, cpp_version, paper_name, paper_link,
		 gcc_support, gcc_display_text, gcc_extra_text,
	     clang_support, clang_display_text, clang_extra_text,
	     msvc_support, msvc_display_text, msvc_extra_text,
	     reported_to_twitter, reported_broken)
		VALUES(:name, :timestamp, :cpp_version, :paper_name, :paper_link,
		 :gcc_support, :gcc_display_text, :gcc_extra_text,
		 :clang_support, :clang_display_text, :clang_extra_text,
		 :msvc_support, :msvc_display_text, :msvc_extra_text,
		 :reported_to_twitter, :reported_broken)`

	//fill automatic fields
	feature.Timestamp = time.Now()
	feature.ReportedToTwitter = false
	feature.ReportedBroken = false

	tx, err := s.db.Beginx()

	if err != nil {
		return errors.Wrap(err, "Failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.NamedExecContext(ctx, query, feature); err != nil {
		return errors.Wrap(err, "failed to insert feature")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "Failed to commit transaction")
	}

	return nil
}
func (s *SqliteService) GetLastIfDiffers(ctx context.Context, feature *Feature) (differs bool, last *Feature, err error) {
	return true, nil, nil
}

//func (s *SqliteService) Create(ctx context.Context, dog *Dog) error {
//	query := `INSERT INTO dogs
//		(name, age, breed, petted_count)
//		VALUES(:name, :age, :breed, :petted_count)`
//
//	tx, err := s.db.Beginx()
//
//	if err != nil {
//		return errors.Wrap(err, "Failed to begin transaction")
//	}
//	defer tx.Rollback()
//
//	if _, err := tx.NamedExecContext(ctx, query, dog); err != nil {
//		return errors.Wrap(err, "failed to insert dog")
//	}
//
//	err = tx.Commit()
//	if err != nil {
//		return errors.Wrap(err, "Failed to commit transaction")
//	}
//
//	return nil
//}
//
//func (s *SqliteService) Get(ctx context.Context, id uint64) (*Dog, error) {
//	query := "SELECT id, name, age, breed, petted_count FROM dogs WHERE id=?"
//
//	dog := &Dog{}
//
//	if err := s.db.Get(dog, query, id); err != nil {
//		return nil, errors.Wrap(err, "Failed to get dog")
//	}
//
//	return dog, nil
//}
//
//func (s *SqliteService) List(ctx context.Context) (Dogs, error) {
//	query := "SELECT id, name, age, breed, petted_count FROM dogs"
//
//	tx, err := s.db.Beginx()
//	if err != nil {
//		return nil, errors.Wrap(err, "Failed to begin transaction")
//	}
//	defer tx.Rollback()
//
//	rows, err := tx.QueryxContext(ctx, query)
//	if err != nil {
//		return nil, err
//	}
//	defer rows.Close()
//
//	result := make(Dogs, 0)
//
//	for rows.Next() {
//		var dog Dog
//		if err := rows.StructScan(&dog); err != nil {
//			return nil, err
//		}
//		result = append(result, &dog)
//	}
//
//	if err = tx.Commit(); err != nil {
//		return nil, errors.Wrap(err, "Failed to commit transaction")
//	}
//
//	return result, nil
//}
//
//func (s *SqliteService) Update(ctx context.Context, dog *Dog) error {
//	query := "UPDATE dogs SET name=:name, age=:age, breed=:breed, petted_count=:petted_count WHERE id=:id"
//
//	tx, err := s.db.Beginx()
//	if err != nil {
//		return errors.Wrap(err, "Failed to begin transaction")
//	}
//	defer tx.Rollback()
//
//	if _, err := tx.NamedExecContext(ctx, query, dog); err != nil {
//		return errors.Wrap(err, "failed to update dog")
//	}
//
//	err = tx.Commit()
//	if err != nil {
//		return errors.Wrap(err, "Failed to commit transaction")
//	}
//
//	return nil
//}
//
//func (s *SqliteService) Delete(ctx context.Context, dog *Dog) error {
//	query := "DELETE FROM dogs WHERE id=?"
//
//	tx, err := s.db.Beginx()
//	if err != nil {
//		return errors.Wrap(err, "Failed to begin transaction")
//	}
//	defer tx.Rollback()
//
//	if _, err := tx.ExecContext(ctx, query, dog.ID); err != nil {
//		return errors.Wrap(err, "failed to delete dog")
//	}
//
//	if err := tx.Commit(); err != nil {
//		return errors.Wrap(err, "Failed to commit transaction")
//	}
//
//	return nil
//}

func (s *SqliteService) Close(ctx context.Context) error {
	return nil
}
