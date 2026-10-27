package db

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOrderRepo_ListOrdersByUserId(t *testing.T) {
	// 1. In-memory БД (никаких файлов .db на диске)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("не удалось открыть БД: %v", err)
	}
	defer db.Close()

	// 2. Создаём схему таблицы ТОЧНО как в твоём проекте
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			item TEXT,
			quantity INTEGER,
			customer TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("ошибка создания таблицы: %v", err)
	}

	// 3. Вставляем тестовые данные НАПРЯМУЮ через SQL (чтобы не зависеть от отсутствующего CreateOrder)
	testOrders := []struct {
		UserID   int
		Status   string
		Item     string
		Quantity int
		Customer string
	}{
		{UserID: 1, Status: "new", Item: "Book", Quantity: 2, Customer: "Alice"},
		{UserID: 1, Status: "reject", Item: "Pen", Quantity: 5, Customer: "Bob"},
		{UserID: 2, Status: "new", Item: "Notebook", Quantity: 1, Customer: "Charlie"},
	}

	for _, o := range testOrders {
		_, err := db.Exec(
			"INSERT INTO orders (user_id, status, item, quantity, customer) VALUES (?, ?, ?, ?, ?)",
			o.UserID, o.Status, o.Item, o.Quantity, o.Customer,
		)
		if err != nil {
			t.Fatalf("не удалось вставить тестовый заказ: %v", err)
		}
	}

	ctx := context.Background()
	repo := NewOrderRepo(db, nil) // hub не нужен для этого теста

	// 4. Проверяем метод, который у тебя ТОЧНО есть: ListOrdersByUserId
	orders, err := repo.ListOrdersByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("не удалось получить заказы: %v", err)
	}

	if len(orders) != 2 {
		t.Errorf("ожидали 2 заказа для user_id=1, получили %d", len(orders))
	}

	// Проверяем конкретные поля
	foundNew := false
	foundReject := false
	for _, o := range orders {
		if o.Status == "new" && o.Item == "Book" {
			foundNew = true
		}
		if o.Status == "reject" && o.Item == "Pen" {
			foundReject = true
		}
	}

	if !foundNew {
		t.Error("не найден заказ со статусом 'new' и товаром 'Book'")
	}
	if !foundReject {
		t.Error("не найден заказ со статусом 'reject' и товаром 'Pen'")
	}
}
