package editor

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kijimaD/ruins/internal/raw"
)

// recipeItem はテンプレートに渡すレシピ行データ
type recipeItem struct {
	Index  int
	Recipe raw.Recipe
	Active bool
}

// recipeEditData はレシピ編集テンプレートに渡すデータ
type recipeEditData struct {
	Index       int
	Recipe      raw.Recipe
	ItemOptions []itemSelectOption
}

// recipesData はレシピ一覧テンプレートに渡すデータ
type recipesData struct {
	Items []recipeItem
	Edit  *recipeEditData
}

func (s *Server) handleRecipes(w http.ResponseWriter, r *http.Request) {
	selected := parseSelectedIndex(r)
	s.renderRecipes(w, selected)
}

func (s *Server) renderRecipes(w http.ResponseWriter, activeIndex int) {
	recipes := s.store.Recipes()
	rows := make([]recipeItem, len(recipes))
	for i, r := range recipes {
		rows[i] = recipeItem{Index: i, Recipe: r, Active: i == activeIndex}
	}
	data := recipesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(recipes) {
		data.Edit = &recipeEditData{
			Index:       activeIndex,
			Recipe:      recipes[activeIndex],
			ItemOptions: s.itemOptions(),
		}
	}
	if err := s.templates.ExecuteTemplate(w, "recipes", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findRecipeIndex(name string) int {
	for i, r := range s.store.Recipes() {
		if r.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleRecipeUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	recipe := parseRecipeForm(r)
	if err := s.store.UpdateRecipe(index, recipe); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/recipes?selected=%d", s.findRecipeIndex(recipe.Name)), http.StatusSeeOther)
}

func (s *Server) handleRecipeCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	recipe := raw.Recipe{Name: name}
	if err := s.store.AddRecipe(recipe); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/recipes?selected=%d", s.findRecipeIndex(name)), http.StatusSeeOther)
}

func (s *Server) handleRecipeDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteRecipe(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/recipes", http.StatusSeeOther)
}

// parseRecipeForm はHTTPフォームからRecipe構造体を構築する
func parseRecipeForm(r *http.Request) raw.Recipe {
	recipe := raw.Recipe{
		Name: r.FormValue("name"),
	}
	// 素材は input_name_0, input_amount_0, input_name_1, ... の形式
	for i := 0; ; i++ {
		name := strings.TrimSpace(r.FormValue(fmt.Sprintf("input_name_%d", i)))
		if name == "" {
			break
		}
		amount, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("input_amount_%d", i)))
		if amount <= 0 {
			amount = 1
		}
		recipe.Inputs = append(recipe.Inputs, raw.RecipeInput{Name: name, Amount: amount})
	}
	return recipe
}
