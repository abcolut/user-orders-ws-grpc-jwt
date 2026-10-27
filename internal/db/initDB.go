package db

import (
	"database/sql"
	"fmt"
	"log"
)

func InitDB(dbPath string) (*sql.DB, error) {

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть БД: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка соединения: %w", err)
	}

	err = InitOrdersDB(db)
	if err != nil {
		db.Close()
		// return fmt.Errorf("ошибка создания таблицы ORDERS: %w", err)
	}

	err = InitUsersDB(db)
	if err != nil {
		db.Close()
		// return fmt.Errorf("ошибка создания таблицы ORDERS: %w", err)
	}

	log.Printf("БД ORDERS & USERS инициализирована: %s", dbPath)

	return db, nil
}
