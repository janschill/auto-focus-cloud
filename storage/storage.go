package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"auto-focus.app/cloud/models"
)

type Database map[string]models.Customer
type CustomerList []models.Customer

type Storage interface {
	GetCustomer(id string) (*models.Customer, error)
	SaveCustomer(customer *models.Customer) error

	FindCustomerByLicenseKey(key string) (*models.Customer, error)

	Close() error
}

type MemoryStorage struct {
	Data Database
}

func (m *MemoryStorage) GetCustomer(id string) (*models.Customer, error) {
	customer, exists := m.Data[id]
	if !exists {
		return nil, nil
	}
	return &customer, nil
}

func (m *MemoryStorage) SaveCustomer(customer *models.Customer) error {
	m.Data[customer.Id] = *customer
	return nil
}

func (m *MemoryStorage) FindCustomerByLicenseKey(key string) (*models.Customer, error) {
	for _, customer := range m.Data {
		for _, license := range customer.Licenses {
			if license.Key == key {
				return &customer, nil
			}
		}
	}
	return nil, nil
}
func (m *MemoryStorage) Close() error {
	return nil
}

type FileStorage struct {
	filepath  string
	customers Database
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
			log.Printf("File %s does not exist, starting with empty database", f.filepath)
			f.customers = make(Database)
			return nil
		}
		return err
	}
	defer file.Close()

	var customers CustomerList
	err = json.NewDecoder(file).Decode(&customers)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	f.customers = make(Database)
	for _, customer := range customers {
		f.customers[customer.Id] = customer
	}

	return nil
}

func (f *FileStorage) GetCustomer(id string) (*models.Customer, error) {
	customer, exists := f.customers[id]
	if !exists {
		return nil, nil
	}
	return &customer, nil
}

func (f *FileStorage) SaveCustomer(customer *models.Customer) error {
	f.customers[customer.Id] = *customer
	// TODO: Write back to file
	return nil
}

func (f *FileStorage) FindCustomerByLicenseKey(key string) (*models.Customer, error) {
	for _, customer := range f.customers {
		for _, license := range customer.Licenses {
			if license.Key == key {
				return &customer, nil
			}
		}
	}
	return nil, nil
}

func (f *FileStorage) Close() error {
	return nil
}
