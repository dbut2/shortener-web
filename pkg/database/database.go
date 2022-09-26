package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dbut2/shortener/pkg/models"
	"github.com/dbut2/shortener/pkg/secrets"
	"github.com/dbut2/shortener/pkg/store"
	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	secrets.GsmResourceID `yaml:"gsmResourceID"`
	Hostname              string `yaml:"hostname"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	Database              string `yaml:"database"`
}

type Database struct {
	db *sql.DB
}

var _ store.Store = new(Database)

func NewDatabase(c Config) (*Database, error) {
	err := secrets.LoadSecret(&c)
	if err != nil {
		return nil, err
	}

	connStr := fmt.Sprintf("%s:%s@(%s)/%s?parseTime=true", c.Username, c.Password, c.Hostname, c.Database)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, err
	}

	return &Database{
		db: db,
	}, nil
}

func (d Database) Set(ctx context.Context, link models.Link) error {
	stmt, err := d.db.PrepareContext(ctx, "INSERT INTO links (code, url, expiry, ip) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}

	dbl := dbLink{
		code: link.Code,
		url:  link.Url,
		expiry: sql.NullTime{
			Time:  link.Expiry.Value,
			Valid: link.Expiry.Valid,
		},
		ip: sql.NullString{
			String: link.IP.Value,
			Valid:  link.IP.Valid,
		},
	}

	res, err := stmt.Exec(dbl.code, dbl.url, dbl.expiry, dbl.ip)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		return errors.New("not 1 row affected")
	}

	return nil
}

func (d Database) Get(ctx context.Context, code string) (models.Link, bool, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT code, url, expiry, ip FROM links WHERE code = ?", code)
	if err != nil {
		return models.Link{}, false, err
	}

	if !rows.Next() {
		return models.Link{}, false, nil
	}

	var dbl dbLink
	err = rows.Scan(&dbl.code, &dbl.url, &dbl.expiry, &dbl.ip)
	if err != nil {
		return models.Link{}, false, err
	}

	link := models.Link{
		Code: dbl.code,
		Url:  dbl.url,
		Expiry: models.NullTime{
			Valid: dbl.expiry.Valid,
			Value: dbl.expiry.Time,
		},
		IP: models.NullString{
			Valid: dbl.ip.Valid,
			Value: dbl.ip.String,
		},
	}

	return link, true, nil
}

type dbLink struct {
	code   string
	url    string
	expiry sql.NullTime
	ip     sql.NullString
}