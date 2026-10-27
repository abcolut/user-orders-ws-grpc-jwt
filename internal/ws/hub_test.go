package ws

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	// "github.com/gorilla/websocket"
)

// ---------- Мок соединения ----------

type mockConn struct {
	mu        sync.Mutex
	messages  [][]byte
	closed    bool
	failWrite bool
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWrite {
		return errors.New("write failed")
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.messages = append(m.messages, cp)
	return nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// ---------- Тесты ----------

func TestNewHub_Empty(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub вернул nil")
	}
	if len(h.rooms) != 0 {
		t.Fatalf("ожидалось 0 комнат, получено %d", len(h.rooms))
	}
}

func TestAddClient(t *testing.T) {
	h := NewHub()
	c := &Client{Conn: &mockConn{}, UserID: 1}

	h.AddClient(1, c)

	clients, ok := h.rooms[1]
	if !ok {
		t.Fatal("комната 1 не создана")
	}
	if len(clients) != 1 || clients[0] != c {
		t.Fatalf("клиент не добавлен в комнату")
	}
}

func TestAddMultipleClientsSameUser(t *testing.T) {
	h := NewHub()
	c1 := &Client{Conn: &mockConn{}, UserID: 1}
	c2 := &Client{Conn: &mockConn{}, UserID: 1}

	h.AddClient(1, c1)
	h.AddClient(1, c2)

	if got := len(h.rooms[1]); got != 2 {
		t.Fatalf("ожидалось 2 клиента, получено %d", got)
	}
}

func TestRemoveClient(t *testing.T) {
	h := NewHub()
	c1 := &Client{Conn: &mockConn{}, UserID: 1}
	c2 := &Client{Conn: &mockConn{}, UserID: 1}

	h.AddClient(1, c1)
	h.AddClient(1, c2)

	h.RemoveClient(1, c1)

	clients := h.rooms[1]
	if len(clients) != 1 || clients[0] != c2 {
		t.Fatalf("после удаления ожидался только c2, получено %v", clients)
	}
}

func TestRemoveClient_RemovesRoomWhenEmpty(t *testing.T) {
	h := NewHub()
	c := &Client{Conn: &mockConn{}, UserID: 1}
	h.AddClient(1, c)

	h.RemoveClient(1, c)

	if _, ok := h.rooms[1]; ok {
		t.Fatal("комната должна быть удалена, когда пуста")
	}
}

func TestRemoveClient_NonExistentUser(t *testing.T) {
	h := NewHub()
	c := &Client{Conn: &mockConn{}, UserID: 42}

	// не должно паниковать
	h.RemoveClient(999, c)
}

func TestRemoveClient_NonExistentClient(t *testing.T) {
	h := NewHub()
	c1 := &Client{Conn: &mockConn{}, UserID: 1}
	c2 := &Client{Conn: &mockConn{}, UserID: 1}

	h.AddClient(1, c1)
	h.RemoveClient(1, c2) // c2 в комнате нет

	if len(h.rooms[1]) != 1 || h.rooms[1][0] != c1 {
		t.Fatalf("удаление несуществующего клиента не должно менять комнату")
	}
}

func TestSendToUser_NoClients(t *testing.T) {
	h := NewHub()
	// не должно паниковать
	h.SendToUser(42, map[string]interface{}{"msg": "hi"})
}

func TestSendToUser_DeliversJSON(t *testing.T) {
	h := NewHub()
	m := &mockConn{}
	c := &Client{Conn: m, UserID: 1}
	h.AddClient(1, c)

	payload := map[string]interface{}{"msg": "hello", "code": float64(200)}
	h.SendToUser(1, payload)

	if len(m.messages) != 1 {
		t.Fatalf("ожидалось 1 сообщение, получено %d", len(m.messages))
	}

	var got map[string]interface{}
	if err := json.Unmarshal(m.messages[0], &got); err != nil {
		t.Fatalf("невалидный JSON: %v", err)
	}
	if got["msg"] != "hello" {
		t.Fatalf("неверное поле msg: %v", got["msg"])
	}
	if got["code"].(float64) != 200 {
		t.Fatalf("неверное поле code: %v", got["code"])
	}
}

func TestSendToUser_MultipleClients(t *testing.T) {
	h := NewHub()
	m1, m2 := &mockConn{}, &mockConn{}
	h.AddClient(1, &Client{Conn: m1, UserID: 1})
	h.AddClient(1, &Client{Conn: m2, UserID: 1})

	h.SendToUser(1, map[string]interface{}{"x": 1})

	if len(m1.messages) != 1 || len(m2.messages) != 1 {
		t.Fatal("оба клиента должны получить сообщение")
	}
}

func TestSendToUser_OnlyTargetRoom(t *testing.T) {
	h := NewHub()
	m1, m2 := &mockConn{}, &mockConn{}
	h.AddClient(1, &Client{Conn: m1, UserID: 1})
	h.AddClient(2, &Client{Conn: m2, UserID: 2})

	h.SendToUser(1, map[string]interface{}{"msg": "for user 1"})

	if len(m1.messages) != 1 {
		t.Fatal("клиент комнаты 1 должен получить сообщение")
	}
	if len(m2.messages) != 0 {
		t.Fatal("клиент комнаты 2 НЕ должен получить сообщение")
	}
}

func TestSendToUser_WriteErrorRemovesClient(t *testing.T) {
	h := NewHub()
	bad := &mockConn{failWrite: true}
	good := &mockConn{}

	badClient := &Client{Conn: bad, UserID: 1}
	goodClient := &Client{Conn: good, UserID: 1}
	h.AddClient(1, badClient)
	h.AddClient(1, goodClient)

	h.SendToUser(1, map[string]interface{}{"msg": "x"})

	// плохой клиент должен быть закрыт и удалён
	if !bad.closed {
		t.Fatal("соединение с ошибкой записи должно быть закрыто")
	}

	clients := h.rooms[1]
	if len(clients) != 1 || clients[0] != goodClient {
		t.Fatalf("после ошибки записи в комнате должен остаться только goodClient, получено %v", clients)
	}
}

func TestSendToUser_Concurrent(t *testing.T) {
	h := NewHub()

	const n = 50
	conns := make([]*mockConn, n)
	for i := 0; i < n; i++ {
		conns[i] = &mockConn{}
		h.AddClient(1, &Client{Conn: conns[i], UserID: 1})
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h.SendToUser(1, map[string]interface{}{"n": i})
		}(i)
	}
	wg.Wait()

	for i, m := range conns {
		if len(m.messages) != 10 {
			t.Fatalf("клиент %d получил %d сообщений, ожидалось 10", i, len(m.messages))
		}
	}
}

// Проверка, что при ошибке маршалинга ничего не отправляется.
// (map[string]interface{} с каналом внутри не сериализуется)
func TestSendToUser_MarshalError(t *testing.T) {
	h := NewHub()
	m := &mockConn{}
	h.AddClient(1, &Client{Conn: m, UserID: 1})

	h.SendToUser(1, map[string]interface{}{"bad": make(chan int)})

	if len(m.messages) != 0 {
		t.Fatal("при ошибке маршалинга сообщение не должно отправляться")
	}
}
