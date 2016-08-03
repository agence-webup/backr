package state

import (
	"encoding/json"
	"fmt"
	"webup/backoops/domain"

	"golang.org/x/net/context"

	"webup/backoops/options"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
)

// BoltStorage implements the Storer interface to store the state locally, using BoltDB
type BoltStorage struct {
	bucket []byte
}

var db *bolt.DB

// NewBoltStorage returns a Storer binded to BoltDB
func NewBoltStorage(opts options.Options) (*BoltStorage, error) {

	if db == nil {
		log.Debugln("Opening BoltDB...")
		newConnection, err := bolt.Open("state.db", 0644, nil)
		if err != nil {
			return nil, err
		}
		db = newConnection
		log.Debugln("BoltDB opened.")
	}

	return &BoltStorage{
		// db:     db,
		bucket: []byte(opts.BackupRootDir),
	}, nil
}

// ConfiguredProjects returns the configured projects (Storer interface)
func (b *BoltStorage) ConfiguredProjects(ctx context.Context) (map[string]domain.Project, error) {

	log.Debugln("Fetching current projects from BoltDB...")

	projects := map[string]domain.Project{}

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
			project := getProjectFromJSON(string(value))

			projects[string(key)] = project
		}

		return nil
	})

	return projects, err
}

// SaveProject store a project (Storer interface)
func (b *BoltStorage) SaveProject(ctx context.Context, project domain.Project) error {

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
func (b *BoltStorage) DeleteProject(ctx context.Context, project domain.Project) error {

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
func (b *BoltStorage) GetProject(ctx context.Context, name string) (*domain.Project, error) {

	var project *domain.Project

	// retrieve the data
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			// just return a nil error, to return an empty projects map without error
			return nil
		}

		value := bucket.Get([]byte(name))
		*project = getProjectFromJSON(string(value))

		return nil
	})

	return project, err
}

// CleanUp cleans the state storage (Storer interface)
func (b *BoltStorage) CleanUp() {
	log.Debugln("Closing BoltDB")
	db.Close()
	db = nil
}
