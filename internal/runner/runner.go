package runner

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
	"hseSQL/internal"
	"hseSQL/internal/database"
	"net/http"
	"strconv"
)

type Runner struct {
	do     *database.DbOperator
	server *http.Server
	router *chi.Mux
}

func NewRunner(config *Config) (*Runner, error) {
	cs, err := database.NewConnectionService(config.DbConfig)
	if err != nil {
		return nil, err
	}
	do := database.NewDbOperator(cs)
	if err := do.CreateTables(); err != nil {
		return nil, err
	}
	r := &Runner{
		do: do,
	}
	r.AddRouter()
	r.server = &http.Server{
		Addr:    config.ServerAddr,
		Handler: r.router,
	}
	return r, nil
}

func (r *Runner) AddRouter() {
	router := chi.NewRouter()
	router.Post("/ei", r.AddEi)
	router.Get("/ei", r.GetEi)

	router.Post("/valuetype", r.AddVT)
	router.Get("/valuetype", r.GetVT)

	router.Post("/class", r.AddC)
	router.Get("/class", r.GetC)
	router.Get("/classtree", r.GetCTree)
	router.Get("/classchildren", r.GetCChildren)
	router.Delete("/class", r.DeleteC)

	router.Post("/product", r.AddP)
	router.Get("/product", r.GetP)
	router.Get("/productclass", r.GetPC)
	router.Put("/product", r.UpdateP)
	router.Delete("/product", r.DeletePC)
	r.router = router
}

func (r *Runner) Run() {
	fmt.Println("starting")
	if err := r.server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (r *Runner) AddEi(w http.ResponseWriter, req *http.Request) {
	var re []*internal.EI
	if err := json.NewDecoder(req.Body).Decode(&re); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ids, err := r.do.CreateAndReadEIs(re)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type response struct {
		Ids []int
	}
	res := response{ids}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) GetEi(w http.ResponseWriter, req *http.Request) {
	eiName := req.URL.Query().Get("ei_name")
	eis, err := r.do.ReadEI(eiName)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := json.NewEncoder(w).Encode(eis); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) AddVT(w http.ResponseWriter, req *http.Request) {
	type request struct {
		ValueTypes []string `json:"value_types"`
	}
	re := &request{}
	if err := json.NewDecoder(req.Body).Decode(&re); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.do.CreateValueTypes(re.ValueTypes); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) GetVT(w http.ResponseWriter, req *http.Request) {
	vts, err := r.do.ReadValueTypes()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type response struct {
		ValueTypes []string `json:"value_types"`
	}
	res := &response{ValueTypes:vts}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) AddC(w http.ResponseWriter, req *http.Request) {
	type request struct {
		Classes []*internal.Class `json:"classes"`
	}
	re := &request{}
	if err := json.NewDecoder(req.Body).Decode(&re); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.do.CreateClasses(re.Classes); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) GetC(w http.ResponseWriter, req *http.Request) {
	idClass := req.URL.Query().Get("class_id")
	id, err := strconv.Atoi(idClass)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	withAllParams := req.URL.Query().Get("all_params")
	wAll, err := strconv.ParseBool(withAllParams)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c, err := r.do.ReadClass(id, wAll)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type response struct {
		Class *internal.Class `json:"class"`
	}
	res := &response{Class:c}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) GetCTree(w http.ResponseWriter, req *http.Request) {
	cc, err := r.do.ReadClassTree()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type response struct {
		Classes []*internal.Class `json:"classes"`
	}
	res := &response{Classes:cc}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) GetCChildren(w http.ResponseWriter, req *http.Request) {
	nameClass := req.URL.Query().Get("class_name")
	c, err := r.do.ReadClassChildren(nameClass)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type response struct {
		Class *internal.Class `json:"class"`
	}
	res := &response{Class:c}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) DeleteC(w http.ResponseWriter, req *http.Request) {
	idClass := req.URL.Query().Get("class_id")
	id, err := strconv.Atoi(idClass)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.do.DeleteClass(id); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) AddP(w http.ResponseWriter, req *http.Request) {
	type request struct {
		Products []*internal.Product `json:"products"`
	}
	re := &request{}
	if err := json.NewDecoder(req.Body).Decode(&re); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.do.CreateProducts(re.Products); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) GetP(w http.ResponseWriter, req *http.Request) {
	idClass := req.URL.Query().Get("product_id")
	id, err := strconv.Atoi(idClass)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p, err := r.do.ReadProduct(id)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type response struct {
		Product *internal.Product `json:"product"`
	}
	res := &response{Product:p}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) GetPC(w http.ResponseWriter, req *http.Request) {
	idClass := req.URL.Query().Get("class_id")
	id, err := strconv.Atoi(idClass)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	pp, err := r.do.ReadClassProducts(id)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	type response struct {
		Products []*internal.Product `json:"products"`
	}
	res := &response{Products:pp}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) UpdateP(w http.ResponseWriter, req *http.Request) {
	type request struct {
		Product *internal.Product `json:"product"`
	}
	re := &request{}
	if err := json.NewDecoder(req.Body).Decode(&re); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.do.UpdateProduct(re.Product); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}

func (r *Runner) DeletePC(w http.ResponseWriter, req *http.Request) {
	idClass := req.URL.Query().Get("product_id")
	id, err := strconv.Atoi(idClass)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.do.DeleteProduct(id); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return
}