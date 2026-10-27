package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"my_api/internal/api"
	dbpkg "my_api/internal/db"

	_ "modernc.org/sqlite"
)

func handleList(w http.ResponseWriter, r *http.Request, repo *dbpkg.OrderRepo, req dbpkg.OrderRequest, currentUserId int) {
	limit := 30
	ctx := r.Context()

	orders, err := dbpkg.ListOrdersByUserId(ctx, repo.DB, limit, currentUserId)
	if err != nil {
		log.Printf("ListOrders error: %v", err)
		api.WriteResponse(w, http.StatusInternalServerError, api.APIResponse{
			Error: &api.APIError{
				Code:    "LIST_ERROR",
				Message: "Ошибка получения списка заказов",
				Errors:  []string{err.Error()},
			},
		})
		return
	}
	api.WriteResponse(w, http.StatusOK, api.APIResponse{Data: orders})
}

func handleAdd(w http.ResponseWriter, repo *dbpkg.OrderRepo, req dbpkg.OrderRequest, currentUserId int) {
	lastId, err := dbpkg.CreateOrder(repo.DB, req, currentUserId)
	if err != nil {
		api.WriteResponse(w, http.StatusInternalServerError, api.APIResponse{
			Error: &api.APIError{
				Code:    "ORDER_CREATION_ERROR",
				Message: "Ошибка создания заказа",
				Errors:  []string{err.Error()},
			},
		})
		return
	}
	resp := dbpkg.OrderResponseAdd{Status: "add", ID: lastId}
	api.WriteResponse(w, http.StatusCreated, api.APIResponse{Data: resp})
}

func handleChangeStatus(w http.ResponseWriter, repo *dbpkg.OrderRepo, req dbpkg.OrderRequest, currentUserId int) {
	// log.Printf("🔄 handleChangeStatus: начинаем обновление статуса для заказа ID=%d", req.ID)

	err := dbpkg.UpdateOrderStatus(repo, req.ID, req.Status, currentUserId)
	if err != nil {
		// log.Printf("❌ handleChangeStatus: ошибка обновления в БД: %v", err)
		api.WriteResponse(w, http.StatusInternalServerError, api.APIResponse{
			Error: &api.APIError{Code: "DB_ERROR", Message: err.Error()},
		})
		return // ВАЖНО: return здесь, дальше не идём
	}

	resp := dbpkg.OrderResponseChangeStatus{Status: "changeStatus", ID: req.ID}
	api.WriteResponse(w, http.StatusOK, api.APIResponse{Data: resp})
}

func handleGetById(w http.ResponseWriter, repo *dbpkg.OrderRepo, req dbpkg.OrderRequest, currentUserId int) {
	id := req.ID
	order, err := dbpkg.GetOrderByID(repo.DB, id, currentUserId)
	if err != nil {
		api.WriteResponse(w, http.StatusInternalServerError, api.APIResponse{
			Error: &api.APIError{
				Code:    "ORDER_READING_ERROR",
				Message: "Ошибка чтения заказа",
				Errors:  []string{err.Error()},
			},
		})
		return
	}
	resp := dbpkg.OrderResponse{
		Item:      order.Item,
		Customer:  order.Customer,
		Quantity:  order.Quantity,
		ID:        order.ID,
		Status:    order.Status,
		CreatedAt: order.CreatedAt,
	}
	api.WriteResponse(w, http.StatusOK, api.APIResponse{Data: resp})
}

func readBodyWithLimit(r *http.Request, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(r.Body, maxBytes)
	return io.ReadAll(limited)
}

func OrdersApiHandler(repo *dbpkg.OrderRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		CurrentUserID, ok := r.Context().Value("userID").(int)
		if !ok {
			api.WriteResponse(w, http.StatusInternalServerError, api.APIResponse{
				Error: &api.APIError{
					Code:    "BODY_ERROR",
					Message: "Ошибка чтения запроса",
					// id : strconv(userID),
				},
			})
			return
		}

		if CurrentUserID < 1 {
			api.WriteResponse(w, http.StatusInternalServerError, api.APIResponse{
				Error: &api.APIError{
					Code:    "BODY_ERROR",
					Message: "User Id < 1",
					// id : strconv(CurrentUserID),
				},
			})
			return
		}

		if r.Method != http.MethodPost {
			api.WriteResponse(w, http.StatusMethodNotAllowed, api.APIResponse{
				Error: &api.APIError{
					Code:    "WRONG_REQUEST",
					Message: "Ошибка POST запроса",
				},
			})
			return
		}

		body, err := readBodyWithLimit(r, 1<<20) // 1 MB лимит
		if err != nil {
			api.WriteResponse(w, http.StatusBadRequest, api.APIResponse{
				Error: &api.APIError{
					Code:    "BODY_ERROR",
					Message: "Ошибка чтения запроса",
				},
			})
			return
		}

		var req dbpkg.OrderRequest
		if err := json.Unmarshal(body, &req); err != nil {
			api.WriteResponse(w, http.StatusBadRequest, api.APIResponse{
				Error: &api.APIError{
					Code:    "JSON_ERROR",
					Message: "Невалидный JSON",
				},
			})
			return
		}

		switch req.Action {
		case "list":
			handleList(w, r, repo, req, CurrentUserID)

		case "changeStatus":
			handleChangeStatus(w, repo, req, CurrentUserID)

		case "getById":
			handleGetById(w, repo, req, CurrentUserID)

		case "add":
			res, _ := api.ValidateOrder(req)

			if len(res) > 0 {
				api.WriteResponse(w, http.StatusBadRequest, api.APIResponse{
					Error: &api.APIError{
						Code:    "VALIDATION_ERROR",
						Message: "Ошибка валидации данных",
						Errors:  res,
					},
				})
				return
			}
			handleAdd(w, repo, req, CurrentUserID)

		default:

			api.WriteResponse(w, http.StatusBadRequest, api.APIResponse{
				Error: &api.APIError{
					Code:    "UNKNOWN_ACTION",
					Message: "Неизвестное действие",
				},
			})
		}
	}
}
