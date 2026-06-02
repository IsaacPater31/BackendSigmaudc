package handlers

import (
	"net/http"
	"strconv"

	"github.com/andrxsq/SIGMAUDC/internal/models"
)

const (
	defaultPage     = 1
	defaultPageSize = 25
	maxPageSize     = 100
)

func parsePagination(r *http.Request, pageKey, pageSizeKey string) (int, int) {
	page := parsePositiveInt(r.URL.Query().Get(pageKey), defaultPage)
	pageSize := parsePositiveInt(r.URL.Query().Get(pageSizeKey), defaultPageSize)

	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func buildPaginationMeta(page, pageSize, totalItems int) models.PaginationMeta {
	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + pageSize - 1) / pageSize
	}
	return models.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    totalPages > 0 && page < totalPages,
		HasPrev:    page > 1 && totalPages > 0,
	}
}

func paginateSlice[T any](items []T, page, pageSize int) ([]T, models.PaginationMeta) {
	total := len(items)
	if total == 0 {
		return []T{}, buildPaginationMeta(page, pageSize, 0)
	}

	start := (page - 1) * pageSize
	if start >= total {
		return []T{}, buildPaginationMeta(page, pageSize, total)
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return items[start:end], buildPaginationMeta(page, pageSize, total)
}
