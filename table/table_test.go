package table

import (
	"errors"
	"fmt"
	"testing"

	"github.com/asdine/genji/engine"
	"github.com/asdine/genji/field"
	"github.com/asdine/genji/record"
	"github.com/stretchr/testify/require"
)

func createTable(t require.TestingT, size int) Reader {
	var rb engine.RecordBuffer

	for i := 0; i < size; i++ {
		rb.Add(record.FieldBuffer{
			field.NewInt64("id", int64(i)),
			field.NewString("name", fmt.Sprintf("john-%d", i)),
			field.NewInt64("age", int64(i*10)),
			field.NewInt64("group", int64(i%3)),
		})
	}

	return NewReader(&rb)
}

func TestReader(t *testing.T) {
	tr := createTable(t, 10)

	t.Run("ForEach", func(t *testing.T) {
		t.Run("Order", func(t *testing.T) {
			i := 0
			tr = tr.ForEach(func(r record.Record) error {
				f, err := r.Field("id")
				require.NoError(t, err)
				v, err := field.DecodeInt64(f.Data)
				require.NoError(t, err)
				require.EqualValues(t, i, v)

				i++
				return nil
			})

			require.NoError(t, tr.Err())
		})

		t.Run("Error", func(t *testing.T) {
			err := errors.New("some error")
			tr2 := tr.ForEach(func(r record.Record) error {
				return err
			})

			// table readers are immutable, tr should not be changed
			require.NoError(t, tr.Err())
			require.Equal(t, err, tr2.Err())
		})
	})

	t.Run("Filter", func(t *testing.T) {
		// filter odd ids
		tr2 := tr.Filter(func(r record.Record) (bool, error) {
			f, err := r.Field("id")
			require.NoError(t, err)
			v, err := field.DecodeInt64(f.Data)
			require.NoError(t, err)
			if v%2 == 0 {
				return true, nil
			}

			return false, nil
		})

		require.NoError(t, tr2.Err())

		t.Run("Immutable", func(t *testing.T) {
			// table readers are immutable, tr should not be changed
			count := 0
			tr := tr.ForEach(func(r record.Record) error {
				count++
				return nil
			})
			require.NoError(t, tr.Err())
			require.Equal(t, 10, count)
		})

		t.Run("OK", func(t *testing.T) {
			// tr2 should only contain even ids
			count := 0
			tr := tr2.ForEach(func(r record.Record) error {
				f, err := r.Field("id")
				require.NoError(t, err)
				v, err := field.DecodeInt64(f.Data)
				require.NoError(t, err)
				require.True(t, v%2 == 0)
				count++
				return nil
			})
			require.NoError(t, tr.Err())
			require.Equal(t, 5, count)
		})

		t.Run("Error", func(t *testing.T) {
			err := errors.New("some error")
			tr2 := tr.Filter(func(r record.Record) (bool, error) {
				return false, err
			})
			require.NoError(t, tr.Err())
			require.Equal(t, err, tr2.Err())
		})
	})

	t.Run("Map", func(t *testing.T) {
		// double the age
		tr2 := tr.Map(func(r record.Record) (record.Record, error) {
			f, err := r.Field("age")
			require.NoError(t, err)
			age, err := field.DecodeInt64(f.Data)
			require.NoError(t, err)

			var fb record.FieldBuffer
			fb.AddFrom(r)

			fb.Set(field.NewInt64("age", age*2))

			return &fb, nil
		})

		require.NoError(t, tr2.Err())

		t.Run("Immutable", func(t *testing.T) {
			// table readers are immutable, tr should not be changed
			i := 0
			tr := tr.ForEach(func(r record.Record) error {
				f, err := r.Field("age")
				require.NoError(t, err)
				age, err := field.DecodeInt64(f.Data)
				require.NoError(t, err)

				require.Equal(t, int64(i*10), age)
				i++
				return nil
			})
			require.NoError(t, tr.Err())
		})

		t.Run("OK", func(t *testing.T) {
			// tr2 ages should be twice as big
			i := 0
			tr := tr2.ForEach(func(r record.Record) error {
				f, err := r.Field("age")
				require.NoError(t, err)
				age, err := field.DecodeInt64(f.Data)
				require.NoError(t, err)

				require.Equal(t, int64(i*20), age)
				i++
				return nil
			})
			require.NoError(t, tr.Err())
		})

		t.Run("Error", func(t *testing.T) {
			err := errors.New("some error")
			tr2 := tr.Map(func(r record.Record) (record.Record, error) {
				return nil, err
			})
			require.NoError(t, tr.Err())
			require.Equal(t, err, tr2.Err())
		})
	})

	t.Run("Count", func(t *testing.T) {
		t.Run("Ok", func(t *testing.T) {
			total, err := tr.Count()
			require.NoError(t, err)
			require.Equal(t, 10, total)
		})
	})

	t.Run("GroupBy", func(t *testing.T) {
		t.Run("Ok", func(t *testing.T) {

			g := tr.GroupBy("group")
			for i, r := range g.Readers {
				r.ForEach(func(r record.Record) error {
					f, err := r.Field("group")
					require.NoError(t, err)

					j, err := field.DecodeInt64(f.Data)
					require.NoError(t, err)

					require.EqualValues(t, i, j)
					return nil
				})
			}

			require.NoError(t, g.Err())
		})
	})

	t.Run("Count", func(t *testing.T) {
		t.Run("Ok", func(t *testing.T) {
			total, err := tr.Count()
			require.NoError(t, err)
			require.Equal(t, 10, total)
		})
	})
}

func TestGroupReader(t *testing.T) {
	g := GroupReader{
		Readers: []Reader{
			createTable(t, 2),
			createTable(t, 3),
		},
	}

	t.Run("Concat", func(t *testing.T) {
		r := g.Concat()
		require.NoError(t, r.Err())
		c, err := r.Count()
		require.NoError(t, err)
		require.Equal(t, 5, c)
	})
}