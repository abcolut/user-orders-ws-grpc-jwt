package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Status    string `json:"status"`
	Token     string `json:"token"`
	CreatedAt string
}

type UserRequest struct {
	Username string `json:"username" validate:"required,min=1,max=20"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=1,max=20"`
}

type UserResponse struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Status    string `json:"status"`
	Token     string `json:"token"`
	CreatedAt string
}

type UserResponseAdd struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type USerResponseChangeStatus struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func ListUsers(ctx context.Context, db *sql.DB, limit int) ([]UserResponse, error) {
	query := `SELECT id, username, email, status, created_at FROM orders ORDER BY id DESC LIMIT ?`
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserResponse
	for rows.Next() {
		var o UserResponse
		err := rows.Scan(
			&o.ID,
			&o.Username,
			&o.Email,
			&o.Status,
			&o.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, o)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func PrintLastUsers(db *sql.DB, limit int) error {
	query := `SELECT id, username, email, status, created_at FROM users ORDER BY id DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("--- последние users ---")
	for rows.Next() {
		var id int
		var username, email, status, createdAt string
		if err := rows.Scan(&id, &username, &email, &status, &createdAt); err != nil {
			return err
		}
		fmt.Printf("ID=%d | %s | %s | %s | %s\n", id, username, email, status, createdAt)
	}
	return nil
}

func UpdateUserStatus(db *sql.DB, id int, newStatus string) error {
	query := `UPDATE users SET status = ? WHERE id = ?`
	res, err := db.Exec(query, newStatus, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления user status: %w", err)
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("user с ID %d не найден для обновления", id)
	}
	return nil
}

func GetUserByID(db *sql.DB, id int) (*UserResponse, error) {
	query := `SELECT id, users, email, status, created_at FROM orders WHERE id = ?`
	row := db.QueryRow(query, id)

	var o UserResponse
	err := row.Scan(
		&o.ID,
		&o.Username,
		&o.Email,
		&o.Status,
		&o.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user с ID %d не найден", id)
		}
		return nil, fmt.Errorf("ошибка чтения карточки user: %w", err)
	}
	return &o, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CreateUser(db *sql.DB, user RegisterRequest) (int, error) {

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		log.Printf("Ошибка хеширования: %v", err)
		return 0, fmt.Errorf("Ошибка хеширования пароля ID: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	query := `INSERT INTO users (username, email, hashed_password, created_at) VALUES (?, ?, ?, ? )`
	res, err := db.Exec(query, user.Username, user.Email, hashedPassword, now)
	if err != nil {
		return 0, fmt.Errorf("ошибка вставки user: %w", err)
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("не удалось получить user ID: %w", err)
	}

	PrintLastOrders(db, 1)
	return int(lastID), nil
}

func InitUsersDB(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL DEFAULT 'guest',
		hashed_password TEXT NOT NULL,
		created_at TEXT NOT NULL
	);`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы ORDERS: %w", err)
	}

	// log.Printf("БД ORDERS инициализирована")
	return nil
}
