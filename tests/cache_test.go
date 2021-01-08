package firestormtests

import (
	"context"
	"errors"
	"github.com/google/go-cmp/cmp"
	"github.com/jschoedt/go-firestorm"
	"github.com/jschoedt/go-firestorm/cache"
	"testing"
	"time"
)

func TestCacheCRUD(t *testing.T) {
	ctx := createSessionCacheContext()
	memoryCache := cache.NewMemoryCache(5*time.Minute, 10*time.Minute)
	fsc.SetCache(memoryCache)

	car := &Car{}
	car.ID = "MyCar"
	car.Make = "Toyota"

	// Create the entity
	fsc.NewRequest().CreateEntities(ctx, car)()
	assertInCache(ctx, memoryCache, car, t)

	// Update the entity
	car.Make = "Jeep"
	fsc.NewRequest().UpdateEntities(ctx, car)()
	m := assertInCache(ctx, memoryCache, car, t)
	if m["make"] != car.Make {
		t.Errorf("Value should be: %v - but was: %v", car.Make, m["Make"])
	}

	// Delete the entity
	fsc.NewRequest().DeleteEntities(ctx, car)()
	if m := assertInCache(ctx, memoryCache, car, t); len(m) > 0 {
		t.Errorf("Value should be: %v - but was: %v", "nil", m)
	}
}

func TestCacheTransaction(t *testing.T) {
	ctx := createSessionCacheContext()
	memoryCache := cache.NewMemoryCache(5*time.Minute, 10*time.Minute)
	fsc.SetCache(memoryCache)

	car := &Car{}
	car.ID = "MyCar"
	car.Make = "Toyota"

	fsc.DoInTransaction(ctx, func(tctx context.Context) error {
		// Create the entity
		fsc.NewRequest().CreateEntities(tctx, car)()
		assertInSessionCache(tctx, car, t)
		assertNotInCache(ctx, memoryCache, car, t)
		return errors.New("rollback")
	})

	assertNotInCache(ctx, memoryCache, car, t)

	fsc.NewRequest().GetEntities(ctx, car)()
	if m := assertInCache(ctx, memoryCache, car, t); m != nil {
		t.Errorf("entity should be nil : %v", m)
	}

	fsc.DoInTransaction(ctx, func(tctx context.Context) error {
		// Create the entity
		return fsc.NewRequest().CreateEntities(tctx, car)()
	})

	if m := assertInCache(ctx, memoryCache, car, t); m == nil {
		t.Errorf("entity should be not be nill : %v", m)
	}

	car.Make = "Jeep"

	fsc.DoInTransaction(ctx, func(tctx context.Context) error {
		// Create the entity
		fsc.NewRequest().UpdateEntities(tctx, car)()
		assertInSessionCache(tctx, car, t)
		assertInCache(ctx, memoryCache, car, t)
		return nil
	})

	assertInCache(ctx, memoryCache, car, t)

	// Delete the entity
	fsc.NewRequest().DeleteEntities(ctx, car)()
	if m := assertInCache(ctx, memoryCache, car, t); len(m) > 0 {
		t.Errorf("Value should be: %v - but was: %v", "nil", m)
	}
}

func assertInSessionCache(ctx context.Context, car *Car, t *testing.T) {
	cacheKey := fsc.NewRequest().ToRef(car).Path
	sessionCache := getSessionCache(ctx)

	if val, ok := sessionCache[cacheKey]; !ok {
		t.Errorf("entity not found in session cache : %v", cacheKey)
		if !cmp.Equal(val, car) {
			t.Errorf("The elements were not the same %v", cmp.Diff(sessionCache[cacheKey], car))
		}
	}
}

func assertInCache(ctx context.Context, memoryCache *cache.InMemoryCache, car *Car, t *testing.T) map[string]interface{} {
	cacheKey := fsc.NewRequest().ToRef(car).Path
	sessionCache := getSessionCache(ctx)
	sesVal, ok := sessionCache[cacheKey]

	assertInSessionCache(ctx, car, t)
	m, err := memoryCache.Get(ctx, cacheKey)
	if err != nil {
		// a nil value was set for a key
		if len(m) == 0 && ok && sesVal == nil {
			return m
		}
		t.Errorf("entity not found in cache : %v", cacheKey)
	}

	if !cmp.Equal(sesVal, m) {
		t.Errorf("The elements were not the same %v", cmp.Diff(sesVal, m))
	}
	return m
}

func assertNotInCache(ctx context.Context, memoryCache *cache.InMemoryCache, car *Car, t *testing.T) {
	cacheKey := fsc.NewRequest().ToRef(car).Path
	sessionCache := getSessionCache(ctx)

	if _, ok := sessionCache[cacheKey]; ok {
		t.Errorf("entity should not be in session cache : %v", cacheKey)
	}

	if _, err := memoryCache.Get(ctx, cacheKey); err != firestorm.ErrCacheMiss {
		t.Errorf("entity should not be in cache : %v", cacheKey)
	}
}
