package impl

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"vcassist-backend/internal/components/assert"
	"vcassist-backend/internal/components/db"
	"vcassist-backend/internal/components/telemetry"

	"github.com/antzucaro/matchr"
)

type Weights struct {
	db  *db.Queries
	tel telemetry.API
}

func NewWeights(db *db.Queries, tel telemetry.API) Weights {
	assert.NotNil(db)
	assert.NotNil(tel)

	tel = telemetry.NewScopedAPI("apis", tel)

	return Weights{db: db, tel: tel}
}

func (w Weights) GetWeights(ctx context.Context, courseId string, categories []string) ([]float32, error) {
	dbCategories, err := w.db.GetWeightCourseCategories(ctx, courseId)
	if err == sql.ErrNoRows {
		w.tel.ReportBroken(report_weights_find_course, courseId, categories)
		return nil, fmt.Errorf("unknown course")
	}
	if err != nil {
		w.tel.ReportBroken(report_db_query, err, "GetWeightCourseCategories", courseId)
		return nil, err
	}

	values := make([]float32, len(categories))
	srcMatched := make([]bool, len(categories))
	dstMatched := make([]bool, len(categories))

	for i, cat := range categories {
		for j, dbCat := range dbCategories {
			if cat == dbCat.CategoryName {
				values[i] = float32(dbCat.Weight)
				srcMatched[i] = true
				dstMatched[j] = true
				break
			}
		}
	}

	type link struct {
		similarity float64
		src        int
		dst        int
	}
	var links []link

	for i, cat := range categories {
		if srcMatched[i] {
			continue
		}
		for j, dbCat := range dbCategories {
			if dstMatched[j] {
				continue
			}
			similarity := matchr.JaroWinkler(cat, dbCat.CategoryName, false)
			links = append(links, link{
				similarity: similarity,
				src:        i,
				dst:        j,
			})
		}
	}

	slices.SortFunc(links, func(a, b link) int {
		// the 1 and -1 are flipped to make it sort descending (large values near the front)
		if a.similarity < b.similarity {
			return 1
		}
		if a.similarity > b.similarity {
			return -1
		}
		return 0
	})

	for _, l := range links {
		if srcMatched[l.src] || dstMatched[l.dst] {
			continue
		}

		w.tel.ReportWarning(
			report_weights_implicit_category_resolution,
			categories[l.src],
			dbCategories[l.dst].CategoryName,
			fmt.Sprintf("weight: %f", dbCategories[l.dst].Weight),
			fmt.Sprintf("similarity: %f", l.similarity),
		)

		srcMatched[l.src] = true
		dstMatched[l.dst] = true
		values[l.src] = float32(dbCategories[l.dst].Weight)
	}

	return values, nil
}

func (w Weights) AddCourse(ctx context.Context, courseId, courseName string) (int64, error) {
	param := db.AddWeightCourseParams{
		ActualCourseID:   courseId,
		ActualCourseName: courseName,
	}
	id, err := w.db.AddWeightCourse(ctx, param)
	if err != nil {
		w.tel.ReportBroken(report_db_query, err, "AddWeightCourse", param)
		return 0, err
	}

	return id, nil
}

func (w Weights) AddCategory(ctx context.Context, weightCourseId int64, categoryName string, weight float64) error {
	param := db.AddWeightCategoryParams{
		WeightCourseID: weightCourseId,
		CategoryName:   categoryName,
		Weight:         weight,
	}
	err := w.db.AddWeightCategory(ctx, param)
	if err != nil {
		w.tel.ReportBroken(report_db_query, err, "AddWeightCategory", param)
		return err
	}
	return nil
}
