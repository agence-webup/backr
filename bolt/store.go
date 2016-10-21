package bolt

import (
	"encoding/json"
	"fmt"
	"time"
	"webup/backr"

	"context"

	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
)

// Storage implements the StateStorage interface to store the state locally, using BoltDB
type Storage struct {
	bucket []byte
}

var db *bolt.DB

func GetStorage(opts backr.Settings) (backr.StateStorer, error) {
	if db == nil {
		log.Debugln("Opening BoltDB...")
		newConnection, err := bolt.Open(filepath.Join(*opts.StateStorage.LocalPath, "state.db"), 0644, &bolt.Options{Timeout: 1 * time.Second})
		if err != nil {
			return nil, err
		}
		db = newConnection
		log.Debugln("BoltDB opened.")
	}

	return &Storage{
		bucket: []byte(opts.BackupRootDir),
	}, nil
}

// Cleanup cleans the opened connection to BoltDB file
func (b *Storage) Cleanup() {
	if db != nil {
		log.Debugln("Closing BoltDB")
		db.Close()
		db = nil
	}
}

// ConfiguredProjects returns the configured projects (Storer interface)
func (b *Storage) ConfiguredProjects(ctx context.Context) (map[string]backr.Project, error) {

	log.Debugln("Fetching current projects from BoltDB...")

	projects := map[string]backr.Project{}

	// retrieve the data
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			log.Debugln("Unable to find the bucket: no project configured. Skipping.")
			// just return a nil error, to return an empty projects map without error
			return nil
		}

		c := bucket.Cursor()

		for key, value := c.First(); key != nil; key, value = c.Next() {
			project := backr.ProjectFromJSON(string(value))

			projects[string(key)] = project
		}

		return nil
	})

	return projects, err
}

// SaveProject store a project (Storer interface)
func (b *Storage) SaveProject(ctx context.Context, project backr.Project) error {

	log.Debugln("Saving a project into BoltDB...")

	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(b.bucket)
		if err != nil {
			log.Debugln("Unable to get or create the bucket into BoltDB.", err)
			return err
		}

		// get json data
		jsonData, _ := json.Marshal(project)

		err = bucket.Put([]byte(project.Name), jsonData)
		if err != nil {
			log.Debugln("Unable to save the project into BoltDB.", err)
			return err
		}
		log.Debugln("Save ok.")
		return nil
	})

	return err
}

// DeleteProject removes a project (Storer interface)
func (b *Storage) DeleteProject(ctx context.Context, project backr.Project) error {

	log.Debugln("Deleting a project from BoltDB...")

	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			log.Debugln("Unable to get or create the bucket into BoltDB.")
			return fmt.Errorf("Bucket not found")
		}

		err := bucket.Delete([]byte(project.Name))
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

// GetProject returns a project (Storer interface)
func (b *Storage) GetProject(ctx context.Context, name string) (*backr.Project, error) {

	var project *backr.Project

	// retrieve the data
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			// just return a nil error, to return an empty projects map without error
			return nil
		}

		value := bucket.Get([]byte(name))
		*project = backr.ProjectFromJSON(string(value))

		return nil
	})

	return project, err
}
