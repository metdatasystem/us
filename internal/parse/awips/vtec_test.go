package internal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/pkg/awips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const awipsTestDataPath = "../../../../data/test/awips/"

func getTestFiles(path string) ([]string, error) {
	dir, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	return files, nil
}

func readFile(t *testing.T, path string, filename string) string {
	file, err := os.ReadFile(path)
	assert.NoErrorf(t, err, "failed to read file %s", filename)
	assert.Greaterf(t, len(file), 0, "%s is nil", filename)
	return string(file)
}

type testSuite struct {
	db    *pgxpool.Pool
	files []string
}

func initTestSuite(t *testing.T, folder string) (*testSuite, error) {
	pool, err := newDatabasePool("postgres://mds:@localhost:5432/mds")
	assert.NoError(t, err, "failed to create database pool")
	assert.NotNil(t, pool, "database pool is nil")

	files, err := getTestFiles(awipsTestDataPath + folder)
	assert.NoError(t, err, "failed to get test files")
	assert.Greater(t, len(files), 0, "no test files found")

	suite := testSuite{
		db:    pool,
		files: files,
	}

	return &suite, nil
}

func (suite *testSuite) teardown() {
	suite.db.Close()
}

func TestTornadoWarning(t *testing.T) {
	dir := "tor/"

	suite, err := initTestSuite(t, dir)
	require.NoError(t, err, "failed to initialize test suite")
	require.NotNil(t, suite, "test suite is nil")
	t.Cleanup(func() {
		defer suite.teardown()

	})

	var year int
	var v *awips.VTEC

	for _, filename := range suite.files {
		text := readFile(t, awipsTestDataPath+dir+filename, filename)

		product, err := awips.New(text)
		assert.NoErrorf(t, err, "failed to parse awips product from file %s", filename)
		assert.NotNilf(t, product, "awips product from file %s is nil", filename)

		HandleText(text, time.Now(), suite.db, nil)

		for _, segments := range product.Segments {
			for _, vtec := range segments.VTEC {
				if year == 0 && vtec.Start != nil {
					year = vtec.Start.Year()
				}
				if v == nil {
					v = &vtec
				}

				// Check VTEC event
				rows, err := suite.db.Query(context.Background(), `
					SELECT * FROM vtec.events WHERE phenomena=$1 AND significance=$2 AND event_number=$3 AND wfo=$4 AND year=$5
					`, vtec.Phenomena, vtec.Significance, vtec.EventNumber, vtec.WFO, year)
				assert.NoError(t, err, "failed to query vtec events")
				assert.True(t, rows.Next(), "no vtec event row returned")

				rows.Close()

			}
		}

	}

	// Cleanup
	_, err = suite.db.Exec(context.Background(), `
		DELETE FROM vtec.events WHERE phenomena=$1 AND significance=$2 AND event_number=$3 AND wfo=$4 AND year=$5
		`, v.Phenomena, v.Significance, v.EventNumber, v.WFO, year)
	assert.NoError(t, err, "failed to delete vtec event")

	_, err = suite.db.Exec(context.Background(), `
		DELETE FROM warnings.warnings WHERE phenomena=$1 AND significance=$2 AND event_number=$3 AND wfo=$4 AND year=$5
		`, v.Phenomena, v.Significance, v.EventNumber, v.WFO, year)
	assert.NoError(t, err, "failed to delete warnings")
}

func TestWinterWeather(t *testing.T) {
	dir := "winter weather/"

	suite, err := initTestSuite(t, dir)
	require.NoError(t, err, "failed to initialize test suite")
	require.NotNil(t, suite, "test suite is nil")
	t.Cleanup(func() {
		defer suite.teardown()

	})

	var year int
	var v *awips.VTEC

	for _, filename := range suite.files {
		text := readFile(t, awipsTestDataPath+dir+filename, filename)

		product, err := awips.New(text)
		assert.NoErrorf(t, err, "failed to parse awips product from file %s", filename)
		assert.NotNilf(t, product, "awips product from file %s is nil", filename)

		HandleText(text, time.Now(), suite.db, nil)

		for _, segments := range product.Segments {
			for _, vtec := range segments.VTEC {
				if year == 0 && vtec.Start != nil {
					year = vtec.Start.Year()
				}
				if v == nil {
					v = &vtec
				}

				// Check VTEC event
				rows, err := suite.db.Query(context.Background(), `
					SELECT * FROM vtec.events WHERE phenomena=$1 AND significance=$2 AND event_number=$3 AND wfo=$4 AND year=$5
					`, vtec.Phenomena, vtec.Significance, vtec.EventNumber, vtec.WFO, year)
				assert.NoError(t, err, "failed to query vtec events")
				assert.True(t, rows.Next(), "no vtec event row returned")

				rows.Close()

			}
		}

	}

	// Cleanup
	// _, err = suite.db.Exec(context.Background(), `
	// 	DELETE FROM vtec.events WHERE phenomena=$1 AND significance=$2 AND event_number=$3 AND wfo=$4 AND year=$5
	// 	`, v.Phenomena, v.Significance, v.EventNumber, v.WFO, year)
	// assert.NoError(t, err, "failed to delete vtec event")

	// _, err = suite.db.Exec(context.Background(), `
	// 	DELETE FROM warnings.warnings WHERE phenomena=$1 AND significance=$2 AND event_number=$3 AND wfo=$4 AND year=$5
	// 	`, v.Phenomena, v.Significance, v.EventNumber, v.WFO, year)
	// assert.NoError(t, err, "failed to delete warnings")
}
