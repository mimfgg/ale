package db

import (
	"context"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/Sirupsen/logrus"
	"github.com/alde/ale"
	"github.com/alde/ale/config"
	"google.golang.org/api/option"
)

// Datastore is a Google Cloud Datastore implementation of the Database interface
type Datastore struct {
	Client    datastoreInterface
	ctx       context.Context
	namespace string
}

type datastoreInterface interface {
	Put(context.Context, *datastore.Key, interface{}) (*datastore.Key, error)
	Get(context.Context, *datastore.Key, interface{}) error
	Count(context.Context, *datastore.Query) (int, error)
}

// NewDatastore creates a new Datastore database object
func NewDatastore(ctx context.Context, cfg *config.Config) (Database, error) {
	credentials := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	var dsClient *datastore.Client
	var err error
	if credentials == "" {
		dsClient, err = datastore.NewClient(ctx, cfg.Database.Project)
	} else {
		dsClient, err = datastore.NewClient(ctx, cfg.Database.Project, option.WithCredentialsFile(credentials))
	}
	if err != nil {
		return nil, err
	}
	return &Datastore{
		Client:    dsClient,
		ctx:       ctx,
		namespace: cfg.Database.Namespace,
	}, nil
}

func (db *Datastore) makeKey(buildID string) *datastore.Key {
	return &datastore.Key{
		Kind:      "JenkinsBuild",
		Name:      buildID,
		Parent:    nil,
		Namespace: db.namespace,
	}
}

// Put inserts data into the database
func (db *Datastore) Put(data *ale.JenkinsData, buildID string) error {
	key := db.makeKey(buildID)
	entity := &ale.DatastoreEntity{
		Key:   buildID,
		Value: *data,
	}
	_, err := db.Client.Put(db.ctx, key, entity)
	return err
}

// Has verifies the existance of a key
func (db *Datastore) Has(buildID string) (bool, error) {
	key := db.makeKey(buildID)
	query := datastore.
		NewQuery("JenkinsBuild").
		Namespace(db.namespace).
		Filter("__key__ =", key).
		Limit(1) // Key should be unique, so limit to 1
	logrus.WithFields(logrus.Fields{
		"build_id": buildID,
	}).Debug("checking the existance of database entry")
	count, err := db.Client.Count(db.ctx, query)
	if err != nil {
		logrus.WithError(err).WithField("build_id", buildID).Debug("database entry not found")
		return false, err
	}
	if count == 1 {
		logrus.WithFields(logrus.Fields{
			"build_id": buildID,
			"count":    count,
		}).Debug("database entry found")
		return true, err
	}
	logrus.WithFields(logrus.Fields{
		"build_id": buildID,
	}).Debug("database entry not found")
	return false, err
}

// Get retrieves data from the database
func (db *Datastore) Get(buildID string) (*ale.JenkinsData, error) {
	var entity ale.DatastoreEntity
	key := db.makeKey(buildID)
	err := db.Client.Get(db.ctx, key, &entity)
	if err != nil {
		return nil, err
	}

	jdata := entity.Value

	return &jdata, nil
}
