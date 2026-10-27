package db

import (
	"context"
	"database/sql"
	"fmt"
	// "log"
	"time"

	"my_api/internal/ws"
)

var Hub *ws.Hub

type OrderRepo struct {
	DB  *sql.DB
	Hub *ws.Hub
}

type OrderGRPC struct {
	ID       int    `json:"id"`
	UserID   int    `json:"user_id"`
	Customer string `json:"customer"`
	Quantity int    `json:"quantity"`
	Item     string `json:"item"`
	Status   string `json:"status"`
	// CreatedAt time.Time `json:"created_at"`
	CreatedAt string `json:"created_at"`
}

type OrderRequest struct {
	ID       int    `json:"id"`
	Customer string `json:"customer" validate:"required,min=1,max=50"`
	Item     string `json:"item" validate:"required,min=1,max=50"`
	Quantity int    `json:"quantity" validate:"required,min=1"`
	Status   string `json:"status"`
	Action   string `json:"action" validate:"required,min=1,max=10"`
}

type OrderResponse struct {
	ID        int    `json:"id"`
	UserId    int    `json:"user_id"`
	Customer  string `json:"customer"`
	Item      string `json:"item"`
	Quantity  int    `json:"quantity"`
	Status    string `json:"status"`
	CreatedAt string
}

type OrderResponseAdd struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type OrderResponseChangeStatus struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

func (r *OrderRepo) ListOrdersByUserId(ctx context.Context, userID int) ([]OrderGRPC, error) {
	var orders []OrderGRPC
	query := `SELECT id, user_id, status, item, customer, quantity FROM orders WHERE user_id = $1`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o OrderGRPC
		err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.Item, &o.Customer, &o.Quantity)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func NewOrderRepo(db *sql.DB, hub *ws.Hub) *OrderRepo {
	return &OrderRepo{DB: db, Hub: hub}
}

func UpdateOrderStatus(repo *OrderRepo, orderID int, newStatus string, currentUserId int) error {
	query := `UPDATE orders SET status = ? WHERE id = ? AND user_id = ?`
	res, err := repo.DB.Exec(query, newStatus, orderID, currentUserId)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса: %w", err)
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("заказ с ID %d не найден для обновления для пользователя user: %d", orderID, currentUserId)
	}

	repo.Hub.SendToUser(currentUserId, map[string]interface{}{
		"type":     "order_status_changed",
		"order_id": orderID,
		"status":   newStatus,
	})
	return nil
}

func ListOrdersByUserId(ctx context.Context, db *sql.DB, limit int, currentUserId int) ([]OrderResponse, error) {
	query := `SELECT id, user_id, customer, item, quantity, status, created_at FROM orders WHERE user_id = ? ORDER BY id DESC LIMIT ?`
	rows, err := db.QueryContext(ctx, query, currentUserId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []OrderResponse
	for rows.Next() {
		var o OrderResponse
		err := rows.Scan(
			&o.ID,
			&o.UserId,
			&o.Customer,
			&o.Item,
			&o.Quantity,
			&o.Status,
			&o.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func PrintLastOrders(db *sql.DB, limit int) error {
	query := `SELECT id, customer, item, quantity, status, user_id, created_at FROM orders ORDER BY id DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, quantity int
		var customer, item, status, createdAt string
		if err := rows.Scan(&id, &customer, &item, &quantity, &status, &createdAt); err != nil {
			return err
		}
		fmt.Printf("ID=%d | %s | %s x%d | %s | %s\n", id, customer, item, quantity, status, createdAt)
	}
	return nil
}

func GetOrderByID(db *sql.DB, id int, currentUserId int) (*OrderResponse, error) {
	query := `SELECT id, user_id, customer, item, quantity, status, created_at FROM orders WHERE id = ? AND user_id = ?`
	row := db.QueryRow(query, id, currentUserId)

	var o OrderResponse
	err := row.Scan(
		&o.ID,
		&o.UserId,
		&o.Customer,
		&o.Item,
		&o.Quantity,
		&o.Status,
		&o.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("заказ с ID %d не найден для пользователя user: %d", id, currentUserId)
		}
		return nil, fmt.Errorf("ошибка чтения заказа: %w", err)
	}
	return &o, nil
}

func CreateOrder(db *sql.DB, order OrderRequest, currentUserId int) (int, error) {
	now := time.Now().Format(time.RFC3339)
	query := `INSERT INTO orders (user_id, customer, item, quantity, status, created_at) VALUES (?, ?, ?, ?, "new", ?)`
	res, err := db.Exec(query, currentUserId, order.Customer, order.Item, order.Quantity, order.Status, now)
	if err != nil {
		return 0, fmt.Errorf("ошибка вставки заказа: %w", err)
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("не удалось получить ID: %w", err)
	}

	PrintLastOrders(db, 1)
	return int(lastID), nil
}

func InitOrdersDB(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		customer TEXT NOT NULL,
		item TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'new',
		created_at TEXT NOT NULL
	);`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы ORDERS: %w", err)
	}

	// log.Printf("БД ORDERS инициализирована")
	return nil
}
