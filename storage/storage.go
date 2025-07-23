package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"auto-focus.app/cloud/models"
	_ "github.com/mattn/go-sqlite3"
)

type Database map[string]models.Customer
type CustomerList []models.Customer

type Storage interface {
	GetCustomer(ctx context.Context, id string) (*models.Customer, error)
	FindCustomerByEmailAddress(ctx context.Context, emailAddress string) (*models.Customer, error)
	SaveCustomer(ctx context.Context, customer *models.Customer) error

	GetLicense(ctx context.Context, id string) (*models.License, error)
	FindLicenseByKey(ctx context.Context, key string) (*models.License, error)
	FindLicensesByCustomer(ctx context.Context, customerID string) ([]*models.License, error)
	SaveLicense(ctx context.Context, license *models.License) error

	Close() error
}

type MemoryStorage struct {
	Data     Database
	Licenses map[string]models.License // Store licenses separately by ID
}

type FileStorage struct {
	filepath  string
	customers Database
	licenses  map[string]models.License // Store licenses separately by ID
}

type SQLiteStorage struct {
	db   *sql.DB
	path string
}

func (m *MemoryStorage) GetCustomer(ctx context.Context, id string) (*models.Customer, error) {
	customer, exists := m.Data[id]
	if !exists {
		return nil, nil
	}
	return &customer, nil
}

func (m *MemoryStorage) FindCustomerByEmailAddress(ctx context.Context, emailAddress string) (*models.Customer, error) {
	for _, customer := range m.Data {
		if customer.Email == emailAddress {
			return &customer, nil
		}
	}
	return nil, nil
}

func (m *MemoryStorage) SaveCustomer(ctx context.Context, customer *models.Customer) error {
	m.Data[customer.ID] = *customer
	return nil
}

func (m *MemoryStorage) GetLicense(ctx context.Context, id string) (*models.License, error) {
	if m.Licenses == nil {
		m.Licenses = make(map[string]models.License)
	}
	license, exists := m.Licenses[id]
	if !exists {
		return nil, nil
	}
	return &license, nil
}

func (m *MemoryStorage) FindLicenseByKey(ctx context.Context, key string) (*models.License, error) {
	if m.Licenses == nil {
		m.Licenses = make(map[string]models.License)
	}
	for _, license := range m.Licenses {
		if license.Key == key {
			return &license, nil
		}
	}
	return nil, nil
}

func (m *MemoryStorage) FindLicensesByCustomer(ctx context.Context, customerID string) ([]*models.License, error) {
	if m.Licenses == nil {
		m.Licenses = make(map[string]models.License)
	}

	var licenses []*models.License
	for _, license := range m.Licenses {
		if license.CustomerID == customerID {
			licenseCopy := license
			licenses = append(licenses, &licenseCopy)
		}
	}

	return licenses, nil
}

func (m *MemoryStorage) SaveLicense(ctx context.Context, license *models.License) error {
	if m.Licenses == nil {
		m.Licenses = make(map[string]models.License)
	}

	// Verify customer exists
	_, exists := m.Data[license.CustomerID]
	if !exists {
		return fmt.Errorf("customer %s not found", license.CustomerID)
	}

	// Save license by ID
	m.Licenses[license.ID] = *license
	return nil
}
func (m *MemoryStorage) Close() error {
	return nil
}

func NewFileStorage(filepath string) (*FileStorage, error) {
	fs := &FileStorage{
		filepath:  filepath,
		customers: make(Database),
	}
	err := fs.loadFromFile()
	return fs, err
}

func (f *FileStorage) loadFromFile() error {
	file, err := os.Open(f.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("file %s does not exist, starting with empty database", f.filepath)
			f.customers = make(Database)
			return nil
		}
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	var customers CustomerList
	err = json.NewDecoder(file).Decode(&customers)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	f.customers = make(Database)
	for _, customer := range customers {
		f.customers[customer.ID] = customer
	}

	return nil
}

func (f *FileStorage) GetCustomer(ctx context.Context, id string) (*models.Customer, error) {
	customer, exists := f.customers[id]
	if !exists {
		return nil, nil
	}
	return &customer, nil
}

func (f *FileStorage) FindCustomerByEmailAddress(ctx context.Context, emailAddress string) (*models.Customer, error) {
	for _, customer := range f.customers {
		if customer.Email == emailAddress {
			return &customer, nil
		}
	}
	return nil, nil
}

func (f *FileStorage) SaveCustomer(ctx context.Context, customer *models.Customer) error {
	f.customers[customer.ID] = *customer
	// TODO: Write back to file
	return nil
}

func (f *FileStorage) GetLicense(ctx context.Context, id string) (*models.License, error) {
	if f.licenses == nil {
		f.licenses = make(map[string]models.License)
	}
	license, exists := f.licenses[id]
	if !exists {
		return nil, nil
	}
	return &license, nil
}

func (f *FileStorage) FindLicenseByKey(ctx context.Context, key string) (*models.License, error) {
	if f.licenses == nil {
		f.licenses = make(map[string]models.License)
	}
	for _, license := range f.licenses {
		if license.Key == key {
			return &license, nil
		}
	}
	return nil, nil
}

func (f *FileStorage) FindLicensesByCustomer(ctx context.Context, customerID string) ([]*models.License, error) {
	if f.licenses == nil {
		f.licenses = make(map[string]models.License)
	}

	var licenses []*models.License
	for _, license := range f.licenses {
		if license.CustomerID == customerID {
			licenseCopy := license
			licenses = append(licenses, &licenseCopy)
		}
	}

	return licenses, nil
}

func (f *FileStorage) SaveLicense(ctx context.Context, license *models.License) error {
	if f.licenses == nil {
		f.licenses = make(map[string]models.License)
	}

	// Verify customer exists
	_, exists := f.customers[license.CustomerID]
	if !exists {
		return fmt.Errorf("customer %s not found", license.CustomerID)
	}

	// Save license by ID
	f.licenses[license.ID] = *license
	// TODO: Write back to file
	return nil
}

func (f *FileStorage) Close() error {
	return nil
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	storage := &SQLiteStorage{
		db:   db,
		path: path,
	}

	ctx := context.Background()
	err = storage.migrate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return storage, nil
}

func (s *SQLiteStorage) migrate(ctx context.Context) error {
	schema := `
      CREATE TABLE IF NOT EXISTS customers (
          id TEXT PRIMARY KEY,
          email TEXT UNIQUE NOT NULL,
          stripe_customer_id TEXT,
          created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
          updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
      );

      CREATE TABLE IF NOT EXISTS licenses (
          id TEXT PRIMARY KEY,
          key TEXT UNIQUE NOT NULL,
          customer_id TEXT NOT NULL,
          product_id TEXT NOT NULL,
          version TEXT NOT NULL,
          status TEXT NOT NULL,
          stripe_session_id TEXT NOT NULL,
          created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
          updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
          FOREIGN KEY (customer_id) REFERENCES customers(id)
      );
      `

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *SQLiteStorage) GetCustomer(ctx context.Context, id string) (*models.Customer, error) {
	query := `SELECT id, email, stripe_customer_id, created_at, updated_at FROM customers WHERE id = ?`

	var customer models.Customer
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&customer.ID,
		&customer.Email,
		&customer.StripeCustomerID,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func (s *SQLiteStorage) FindCustomerByEmailAddress(ctx context.Context, emailAddress string) (*models.Customer, error) {
	query := `SELECT id, email, stripe_customer_id, created_at, updated_at FROM customers WHERE email = ?`

	var customer models.Customer
	err := s.db.QueryRowContext(ctx, query, emailAddress).Scan(
		&customer.ID,
		&customer.Email,
		&customer.StripeCustomerID,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func (s *SQLiteStorage) SaveCustomer(ctx context.Context, customer *models.Customer) error {
	query := `INSERT OR REPLACE INTO customers (id, email, stripe_customer_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		customer.ID,
		customer.Email,
		customer.StripeCustomerID,
		customer.CreatedAt,
		customer.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save customer: %w", err)
	}

	return nil
}

func (s *SQLiteStorage) GetLicense(ctx context.Context, id string) (*models.License, error) {
	query := `SELECT id, key, customer_id, product_id, version, status, stripe_session_id, created_at, updated_at FROM licenses WHERE id = ?`

	var license models.License
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&license.ID,
		&license.Key,
		&license.CustomerID,
		&license.ProductID,
		&license.Version,
		&license.Status,
		&license.StripeSessionID,
		&license.CreatedAt,
		&license.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &license, nil
}

func (s *SQLiteStorage) FindLicenseByKey(ctx context.Context, key string) (*models.License, error) {
	query := `SELECT id, key, customer_id, product_id, version, status, stripe_session_id, created_at, updated_at FROM licenses WHERE key = ?`

	var license models.License
	err := s.db.QueryRowContext(ctx, query, key).Scan(
		&license.ID,
		&license.Key,
		&license.CustomerID,
		&license.ProductID,
		&license.Version,
		&license.Status,
		&license.StripeSessionID,
		&license.CreatedAt,
		&license.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &license, nil
}

func (s *SQLiteStorage) FindLicensesByCustomer(ctx context.Context, customerID string) ([]*models.License, error) {
	query := `SELECT id, key, customer_id, product_id, version, status, stripe_session_id, created_at, updated_at FROM licenses WHERE customer_id = ?`

	rows, err := s.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query licenses: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var licenses []*models.License

	for rows.Next() {
		var license models.License
		err := rows.Scan(
			&license.ID,
			&license.Key,
			&license.CustomerID,
			&license.ProductID,
			&license.Version,
			&license.Status,
			&license.StripeSessionID,
			&license.CreatedAt,
			&license.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan license: %w", err)
		}

		licenses = append(licenses, &license)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating licenses: %w", err)
	}

	return licenses, nil
}

func (s *SQLiteStorage) SaveLicense(ctx context.Context, license *models.License) error {
	query := `INSERT OR REPLACE INTO licenses (id, key, version, status, customer_id, product_id, stripe_session_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		license.ID,
		license.Key,
		license.Version,
		license.Status,
		license.CustomerID,
		license.ProductID,
		license.StripeSessionID,
		license.CreatedAt,
		license.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save customer: %w", err)
	}

	return nil

}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
