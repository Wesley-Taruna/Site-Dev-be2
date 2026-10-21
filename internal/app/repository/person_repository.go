package repository

import (
	"context"

	"site-dev-backend/global"
	"site-dev-backend/internal/app/model"

	"gitlab.com/fami-app-v2/familib/familabs/opt"
)

type IPersonRepository interface {
	Get(ctx context.Context, offset int, limit int, search string) (persons []*model.Person, count int64, err error)
	GetByID(ctx context.Context, id int64) (person *model.Person, err error)
	Create(ctx context.Context, person *model.Person) (err error)
	Update(ctx context.Context, person *model.Person) (err error)
	Delete(ctx context.Context, person *model.Person) (err error)
}

type PersonRepository struct {
	opt opt.Options
}

func NewPersonRepository() IPersonRepository {
	return &PersonRepository{
		opt: global.Setup.Common,
	}
}

func (r *PersonRepository) Get(ctx context.Context, offset int, limit int, search string) (persons []*model.Person, count int64, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "PersonRepository.Get")
	defer span.End()

	// db := r.opt.Database.Read
	// count, err = db.QueryTable(new(model.Person)).Filter("name__icontains", search).Offset(offset).Limit(limit).AllWithCtx(ctx, &persons)

	if err != nil {
		span.RecordError(err)
		return
	}

	return
}

func (r *PersonRepository) GetByID(ctx context.Context, id int64) (person *model.Person, err error) {
	ctx, span := r.opt.Trace.Start(ctx, "PersonRepository.GetByID")
	defer span.End()

	// caching, err := r.opt.Redisx.Caching()
	// if err != nil {
	// 	span.RecordError(err)
	// 	return
	// }

	// person = &model.Person{
	// 	ID: id,
	// }

	// err = caching.Once(&cache.Item{
	// 	Ctx:   ctx,
	// 	Key:   fmt.Sprintf("person:%d", id),
	// 	Value: person,
	// 	TTL:   15 * time.Minute,
	// 	Do: func(item *cache.Item) (interface{}, error) {
	// 		db := r.opt.Database.Read
	// 		err = db.ReadWithCtx(ctx, person, "id")
	// 		if err != nil {
	// 			span.RecordError(err)
	// 			return nil, err
	// 		}
	// 		return person, nil
	// 	},
	// })
	// if err != nil {
	// 	span.RecordError(err)
	// 	return
	// }

	return
}

func (r *PersonRepository) Create(ctx context.Context, person *model.Person) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "PersonRepository.Create")
	defer span.End()

	// db := r.opt.Database.Write
	// tx, err := db.BeginWithCtx(ctx)
	// if err != nil {
	// 	return
	// }

	// _, err = tx.InsertWithCtx(ctx, person)

	// if err == nil {
	// 	err = tx.Commit()
	// 	if err != nil {
	// 		span.RecordError(err)
	// 		return
	// 	}
	// } else {
	// 	span.RecordError(err)
	// 	err = tx.Rollback()
	// 	if err != nil {
	// 		span.RecordError(err)
	// 		return
	// 	}
	// }

	return
}

func (r *PersonRepository) Update(ctx context.Context, person *model.Person) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "PersonRepository.Update")
	defer span.End()

	// db := r.opt.Database.Write
	// tx, err := db.BeginWithCtx(ctx)
	// if err != nil {
	// 	return
	// }

	// _, err = tx.UpdateWithCtx(ctx, person)

	// if err == nil {
	// 	err = tx.Commit()
	// 	if err != nil {
	// 		span.RecordError(err)
	// 		return
	// 	}
	// } else {
	// 	span.RecordError(err)
	// 	err = tx.Rollback()
	// 	if err != nil {
	// 		span.RecordError(err)
	// 		return
	// 	}
	// }

	return
}

func (r *PersonRepository) Delete(ctx context.Context, person *model.Person) (err error) {
	ctx, span := r.opt.Trace.Start(ctx, "PersonRepository.Update")
	defer span.End()

	// db := r.opt.Database.Write
	// tx, err := db.BeginWithCtx(ctx)
	// if err != nil {
	// 	return
	// }

	// _, err = tx.DeleteWithCtx(ctx, person, "id")

	// if err == nil {
	// 	err = tx.Commit()
	// 	if err != nil {
	// 		span.RecordError(err)
	// 		return
	// 	}
	// } else {
	// 	span.RecordError(err)
	// 	err = tx.Rollback()
	// 	if err != nil {
	// 		span.RecordError(err)
	// 		return
	// 	}
	// }
	return
}
