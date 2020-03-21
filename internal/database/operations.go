package database

import "C"
import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v4"
	"hseSQL/internal"
)

type DbOperator struct {
	cs *ConnectionService
}

func NewDbOperator(cs *ConnectionService) *DbOperator {
	return &DbOperator{
		cs: cs,
	}
}

func (do *DbOperator) CreateTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS EI (
		ID_EI SERIAL PRIMARY KEY,
		NAME VARCHAR(250) UNIQUE CHECK (LENGTH(NAME) > 0),
		SHORT_NAME VARCHAR(20) CHECK (LENGTH(NAME) > 0))`,

		`CREATE TABLE IF NOT EXISTS VALUE_TYPES (
		ID_VALUE_TYPE SERIAL PRIMARY KEY,
		NAME VARCHAR(20) UNIQUE CHECK (LENGTH(NAME) > 0))`,

		`CREATE TABLE IF NOT EXISTS PARAMS (
		ID_PARAM SERIAL PRIMARY KEY,
		NAME VARCHAR(200) UNIQUE CHECK (LENGTH(NAME) > 0),
		ID_VALUE_TYPE INTEGER REFERENCES VALUE_TYPES(ID_VALUE_TYPE) ON DELETE CASCADE,
		ID_EI INTEGER REFERENCES EI(ID_EI) ON DELETE SET DEFAULT)`,

		`CREATE TABLE IF NOT EXISTS CLASSES (
		ID_CLASS SERIAL PRIMARY KEY,
		NAME VARCHAR(300) UNIQUE CHECK (LENGTH(NAME) > 0),
		ID_PARENT_CLASS INTEGER REFERENCES CLASSES(ID_CLASS) ON DELETE CASCADE,
		ID_EI INTEGER REFERENCES EI(ID_EI) ON DELETE SET DEFAULT,
		UNIQUE (ID_CLASS, ID_PARENT_CLASS))`,

		`CREATE TABLE IF NOT EXISTS CLASS_PARAMS (
		ID_CLASS_PARAM SERIAL PRIMARY KEY,
		ID_CLASS INTEGER REFERENCES CLASSES(ID_CLASS) ON DELETE CASCADE,
		ID_PARAM INTEGER REFERENCES PARAMS(ID_PARAM) ON DELETE CASCADE,
		UNIQUE (ID_CLASS, ID_PARAM))`,

		`CREATE TABLE IF NOT EXISTS PRODUCTS (
		ID_PRODUCT SERIAL PRIMARY KEY,
		NAME VARCHAR(300) UNIQUE CHECK (LENGTH(NAME) > 0),
		ID_PARENT_CLASS INTEGER REFERENCES CLASSES(ID_CLASS) ON DELETE CASCADE)`,

		`CREATE TABLE IF NOT EXISTS PRODUCT_PARAM_VALUES (
		ID_PRODUCT INTEGER REFERENCES PRODUCTS(ID_PRODUCT) ON DELETE CASCADE,
		ID_PARAM INTEGER REFERENCES CLASS_PARAMS(ID_CLASS_PARAM) ON DELETE CASCADE,
		VALUE VARCHAR(300),
		UNIQUE (ID_PRODUCT, ID_PARAM))`,
	}
	f := func(tx pgx.Tx) error {
		for _, t := range tables {
			if _, err := tx.Exec(context.Background(), t); err != nil {
				return err
			}
		}
		return nil
	}
	return do.cs.WrapIntoTransaction(context.Background(), f)
}

// EI

func (do *DbOperator) CreateAndReadEIs(eis []*internal.EI) ([]int, error) {
	var ids []int
	f := func(tx pgx.Tx) error {
		for _, ei := range eis {
			eiId, err := do.cr_EI(tx, ei.Name, ei.ShortName)
			if err != nil {
				return err
			}
			ids = append(ids, eiId)
		}
		return nil
	}
	return ids, do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) ReadEI(searchName string) ([]*internal.EI, error) {
	var res []*internal.EI
	f := func(tx pgx.Tx) error {
		eis, err := do.r_EI(tx, searchName)
		if err != nil {
			return err
		}
		res = eis
		return nil
	}
	return res, do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) cr_EI(tx pgx.Tx, name, shortName string) (id int, err error) {
	err = tx.QueryRow(context.Background(),
		`SELECT ID_EI 
			FROM EI 
			WHERE NAME = $1`,
		name).Scan(&id)
	if err != nil {
		err = tx.QueryRow(context.Background(),
			`INSERT INTO EI(NAME, SHORT_NAME) 
				VALUES($1,$2) 
				RETURNING ID_EI`,
			name, shortName).Scan(&id)
	}
	return
}

func (do *DbOperator) r_EI(tx pgx.Tx, searchName string) ([]*internal.EI, error) {
	var rows pgx.Rows
	var err error
	if searchName != "" {
		rows, err = tx.Query(context.Background(),
			`SELECT ID_EI, NAME, SHORT_NAME 
				FROM EI 
				WHERE NAME = $1`,
			searchName)
	} else {
		rows, err = tx.Query(context.Background(),
			`SELECT ID_EI, NAME, SHORT_NAME 
				FROM EI`)
	}
	if err != nil {
		return nil, err
	}
	var result []*internal.EI
	var id int
	var name, shortName string
	for rows.Next() {
		if err := rows.Scan(&id, &name, &shortName); err != nil {
			rows.Close()
			return nil, err
		}
		result = append(result, &internal.EI{
			Id:        id,
			Name:      name,
			ShortName: shortName,
		})
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// VALUE_TYPES

func (do *DbOperator) CreateValueTypes(vts []string) (err error) {
	f := func(tx pgx.Tx) error {
		for _, vt := range vts {
			if err := do.c_ValueType(tx, vt); err != nil {
				return err
			}
		}
		return nil
	}
	return do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) ReadValueTypes() (vts []string, err error) {
	f := func(tx pgx.Tx) error {
		vts, err = do.r_ValueType(tx)
		if err != nil {
			return err
		}
		return nil
	}
	return vts, do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) c_ValueType(tx pgx.Tx, name string) error {
	_, err := tx.Exec(context.Background(),
		`INSERT INTO VALUE_TYPES(NAME)
			VALUES($1)`,
		name)
	return err
}

func (do *DbOperator) r_ValueType(tx pgx.Tx) ([]string, error) {
	rows, err := tx.Query(context.Background(),
		`SELECT NAME 
				FROM VALUE_TYPES`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	var name string
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, name)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// PARAMS

func (do *DbOperator) c_Param(tx pgx.Tx, p *internal.Param) (id int, err error) {
	var idValueType, idEi int
	if err = tx.QueryRow(context.Background(),
		`SELECT ID_VALUE_TYPE 
			FROM VALUE_TYPES 
			WHERE NAME = $1`,
		p.ValType).Scan(&idValueType); err != nil {
		return
	}
	if err = tx.QueryRow(context.Background(),
		`SELECT ID_EI 
			FROM EI 
			WHERE NAME = $1`,
		p.EI.Name).Scan(&idEi); err != nil {
		return
	}
	err = tx.QueryRow(context.Background(),
		`INSERT INTO PARAMS(NAME, ID_VALUE_TYPE, ID_EI) 
			VALUES($1,$2,$3) RETURNING ID_PARAM`,
		p.Name, idValueType, idEi).Scan(&id)
	return
}

// CLASSES

func (do *DbOperator) c_Class(tx pgx.Tx, c *internal.Class, parentClass sql.NullInt32) (id int, err error) {
	// meaning that we didn't have the parent in the beginning of the procedure
	if !(parentClass.Valid && parentClass.Int32 != 0) {
		parentClass, err = do.r_ParentClass(tx, c.Name)
		if err != nil {
			return
		}
	}
	ei, err := do.r_EI(tx, c.Ei.Name)
	if err != nil {
		return
	}
	if len(ei) == 0 {
		return 0, errors.New("couldn't find ei")
	}
	err = tx.QueryRow(context.Background(),
		`INSERT INTO CLASSES(NAME, ID_PARENT_CLASS, ID_EI) 
			VALUES($1,$2,$3) 
			RETURNING ID_CLASS`,
		c.Name, parentClass, ei[0].Id).Scan(&id)
	if err != nil {
		return
	}
	if err = do.c_ClassParams(tx, c); err != nil {
		return
	}
	for _, child := range c.Children {
		_, err = do.c_Class(tx, child, sql.NullInt32{
			Int32: int32(id),
			Valid: true,
		})
		if err != nil {
			return 0, err
		}
	}
	return
}

func (do *DbOperator) r_ClassId(tx pgx.Tx, name string) (id int, err error) {
	err = tx.QueryRow(context.Background(),
		`SELECT ID_CLASS 
			FROM CLASSES 
			WHERE NAME = $1`,
		name).Scan(&id)
	return
}

func (do *DbOperator) r_ParentClass(tx pgx.Tx, name string) (id sql.NullInt32, err error) {
	err = tx.QueryRow(context.Background(),
		`SELECT C_PARENT.ID_CLASS 
			FROM CLASSES C_PARENT RIGHT JOIN CLASSES C_CHILD ON C_PARENT.ID_CLASS = C_CHILD.ID_PARENT_CLASS 
			WHERE C_CHILD.NAME = $1`,
		name).Scan(&id)
	if err == pgx.ErrNoRows {
		return id, nil
	}
	return
}

func (do *DbOperator) r_Class(tx pgx.Tx, idClass int, withParams bool) (*internal.Class, error) {
	var name, eiName, eiShortName string
	if err := tx.QueryRow(context.Background(),
		`SELECT C.NAME, EIC.NAME, EIC.SHORT_NAME
			FROM CLASSES C JOIN EI EIC ON C.ID_EI = EIC.ID_EI
			WHERE C.ID_CLASS = $1`,
		idClass).Scan(&name, &eiName, &eiShortName); err != nil {
		return nil, err
	}
	c := &internal.Class{
		Id:       idClass,
		Name:     name,
		Children: nil,
		Ei: &internal.EI{
			Id:        0,
			Name:      eiName,
			ShortName: eiShortName,
		},
		Params: []*internal.Param{},
	}
	if withParams {
		rows, err := tx.Query(context.Background(),
			`WITH CLASS_FAMILY AS (
				SELECT ID_CLASS, ID_PARENT_CLASS
				FROM CLASSES
				WHERE NAME = $1 UNION
				SELECT * FROM (
					WITH RECURSIVE SUBCLASSES AS (
						SELECT ID_CLASS, ID_PARENT_CLASS 
						FROM CLASSES
						WHERE NAME = $1 UNION
						SELECT C.ID_CLASS, C.ID_PARENT_CLASS 
						FROM CLASSES C INNER JOIN SUBCLASSES S ON S.ID_PARENT_CLASS = C.ID_CLASS) 
					SELECT * FROM SUBCLASSES
				) ITER ORDER BY ID_PARENT_CLASS NULLS FIRST
			)
			SELECT P.ID_PARAM, P.NAME, VT.NAME, EIP.NAME, EIP.SHORT_NAME, CP.ID_CLASS
			FROM CLASS_PARAMS CP JOIN CLASS_FAMILY CF ON CF.ID_CLASS = CP.ID_CLASS
							JOIN PARAMS P ON CP.ID_PARAM = P.ID_PARAM
							JOIN VALUE_TYPES VT ON P.ID_VALUE_TYPE = VT.ID_VALUE_TYPE
							JOIN EI EIP ON P.ID_EI = EIP.ID_EI`,
			c.Name)
		if err != nil {
			return nil, err
		}
		var idParam, idParamOwner int
		var paramName, valTypeName, eiParamName, eiParamShortName string
		for rows.Next() {
			if err = rows.Scan(&idParam, &paramName, &valTypeName, &eiParamName, &eiParamShortName, &idParamOwner); err != nil {
				rows.Close()
				return nil, err
			}
			c.Params = append(c.Params, &internal.Param{
				IdParamOwner: idParamOwner,
				Id:      idParam,
				Name:    paramName,
				ValType: valTypeName,
				EI: &internal.EI{
					Id:        0,
					Name:      eiParamName,
					ShortName: eiParamShortName,
				},
			})
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (do *DbOperator) r_FullClassTree(tx pgx.Tx) ([]*internal.Class, error) {
	rows, err := tx.Query(context.Background(),
		`SELECT ID_CLASS, NAME, ID_PARENT_CLASS
			FROM CLASSES`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roots []*internal.Class
	classes := make(map[int]*internal.Class)
	var idClass, idParent sql.NullInt32
	var name string
	for rows.Next() {
		if err := rows.Scan(&idClass, &name, &idParent); err != nil {
			return nil, err
		}
		c := &internal.Class{
			Id:       int(idClass.Int32),
			Name:     name,
			Children: []*internal.Class{},
		}
		classes[int(idClass.Int32)] = c
		if !idParent.Valid {
			roots = append(roots, c)
			continue
		}
		parent, ok := classes[int(idParent.Int32)]
		if !ok {
			return nil, errors.New("couldn't find parent")
		}
		parent.Children = append(parent.Children, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roots, nil
}

func (do *DbOperator) r_ClassChildren(tx pgx.Tx, searchName string) (*internal.Class, error) {
	rows, err := tx.Query(context.Background(),
		`SELECT ID_CLASS, NAME, ID_PARENT_CLASS
			FROM CLASSES
			WHERE NAME = $1 UNION
			SELECT * FROM (
				WITH RECURSIVE SUBCLASSES AS (
					SELECT ID_CLASS, NAME, ID_PARENT_CLASS 
					FROM CLASSES
					WHERE NAME = $1 UNION
					SELECT C.ID_CLASS, C.NAME, C.ID_PARENT_CLASS 
					FROM CLASSES C INNER JOIN SUBCLASSES S ON S.ID_CLASS = C.ID_PARENT_CLASS) 
				SELECT * FROM SUBCLASSES
			) ITER ORDER BY ID_PARENT_CLASS NULLS FIRST`,
		searchName)
	if err != nil {
		return nil, err
	}
	classes := make(map[int]*internal.Class)
	var initialClass *internal.Class
	var id, parent sql.NullInt32
	var name string
	for rows.Next() {
		if err := rows.Scan(&id, &name, &parent); err != nil {
			rows.Close()
			return nil, err
		}
		class := &internal.Class{
			Id:       int(id.Int32),
			Name:     name,
			Children: []*internal.Class{},
		}
		classes[int(id.Int32)] = class
		if initialClass == nil {
			initialClass = class
			continue
		}
		if parent.Valid {
			c, ok := classes[int(parent.Int32)]
			if !ok {
				return nil, errors.New("couldn't find parent class")
			}
			c.Children = append(c.Children, class)
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return initialClass, nil
}

func (do *DbOperator) u_Class(tx pgx.Tx, c *internal.Class) (id int, err error) {
	if err = do.d_Class(tx, c.Id); err != nil {
		return
	}
	return do.c_Class(tx, c, sql.NullInt32{})
}

func (do *DbOperator) d_Class(tx pgx.Tx, id int) (err error) {
	var idCheck int
	if err = tx.QueryRow(context.Background(),
		`SELECT ID_CLASS
			FROM CLASSES
			WHERE ID_CLASS = $1`,
		id).Scan(&idCheck); err != nil {
		return err
	}
	_, err = tx.Exec(context.Background(),
		`DELETE FROM CLASSES 
			WHERE ID_CLASS = $1`,
		id)
	return
}

func (do *DbOperator) CreateClasses(cc []*internal.Class) (err error) {
	f := func(tx pgx.Tx) error {
		for _, c := range cc {
			_, err := do.c_Class(tx, c, sql.NullInt32{})
			if err != nil {
				return err
			}
		}
		return nil
	}
	return do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) ReadClass(id int, withAllParams bool) (*internal.Class, error) {
	var c *internal.Class
	f := func(tx pgx.Tx) error {
		cl, err := do.r_Class(tx, id, withAllParams)
		if err != nil {
			return err
		}
		c = cl
		return nil
	}
	return c, do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) ReadClassTree() ([]*internal.Class, error) {
	var cc []*internal.Class
	f := func(tx pgx.Tx) error {
		classes, err := do.r_FullClassTree(tx)
		if err != nil {
			return err
		}
		cc = classes
		return nil
	}
	return cc, do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) ReadClassChildren(searchName string) (*internal.Class, error) {
	var c *internal.Class
	f := func(tx pgx.Tx) error {
		cl, err := do.r_ClassChildren(tx, searchName)
		if err != nil {
			return err
		}
		c = cl
		return nil
	}
	return c, do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) DeleteClass(id int) error {
	f := func(tx pgx.Tx) error {
		if err := do.d_Class(tx, id); err != nil {
			return err
		}
		return nil
	}
	return do.cs.WrapIntoTransaction(context.Background(), f)
}

// CLASS_PARAMS

func (do *DbOperator) c_ClassParams(tx pgx.Tx, c *internal.Class) (err error) {
	for _, param := range c.Params {
		idClass, err := do.r_ClassId(tx, c.Name)
		if err != nil {
			return err
		}
		idParam, err := do.c_Param(tx, param)
		if err != nil {
			return err
		}
		_, err = tx.Exec(context.Background(),
			`INSERT INTO CLASS_PARAMS(ID_CLASS, ID_PARAM)
				VALUES($1,$2)`,
			idClass, idParam)
		if err != nil {
			return err
		}
	}
	return
}

// PRODUCTS

func (do *DbOperator) c_Product(tx pgx.Tx, p *internal.Product) (err error) {
	class, err := do.r_ClassChildren(tx, p.ParentClass.Name)
	if err != nil {
		return err
	}
	if class.Children == nil || len(class.Children) != 0 {
		return errors.New("can't add product to non-terminal class")
	}
	_, err = tx.Exec(context.Background(),
		`INSERT INTO PRODUCTS(NAME, ID_PARENT_CLASS) 
			VALUES($1,$2) 
			RETURNING ID_PRODUCT`,
		p.Name, class.Id)
	if err != nil {
		return
	}
	if err = do.c_ProductParams(tx, p); err != nil {
		return
	}
	return
}

func (do *DbOperator) r_ProductId(tx pgx.Tx, name string) (id int, err error) {
	err = tx.QueryRow(context.Background(),
		`SELECT ID_PRODUCT 
			FROM PRODUCTS 
			WHERE NAME = $1`,
		name).Scan(&id)
	return
}

func (do *DbOperator) r_Product(tx pgx.Tx, id int) (*internal.Product, error) {
	var name string
	var idParent int
	if err := tx.QueryRow(context.Background(),
		`SELECT NAME, ID_PARENT_CLASS
			FROM PRODUCTS 
			WHERE ID_PRODUCT = $1`,
		id).Scan(&name, &idParent); err != nil {
		return nil, err
	}
	class, err := do.r_Class(tx, idParent, false)
	if err != nil {
		return nil, err
	}
	p := &internal.Product{
		Id:          id,
		Name:        name,
		ParentClass: class,
		Params:      []*internal.ParamAndValues{},
	}
	rows, err := tx.Query(context.Background(),
		`SELECT P.NAME, VT.NAME, EI.NAME, EI.SHORT_NAME, PPV.VALUE
			FROM PRODUCT_PARAM_VALUES PPV JOIN CLASS_PARAMS CP ON PPV.ID_PARAM = CP.ID_CLASS_PARAM
										JOIN PARAMS P ON P.ID_PARAM = CP.ID_PARAM
										JOIN VALUE_TYPES VT ON P.ID_VALUE_TYPE = VT.ID_VALUE_TYPE
										JOIN EI ON EI.ID_EI = P.ID_EI
			WHERE PPV.ID_PRODUCT = $1`,
		id)
	if err != nil {
		return nil, err
	}
	var paramName, paramValueType, paramEiName, paramEiShortName, value string
	for rows.Next() {
		if err = rows.Scan(&paramName, &paramValueType, &paramEiName, &paramEiShortName, &value); err != nil {
			rows.Close()
			return nil, err
		}
		p.Params = append(p.Params, &internal.ParamAndValues{
			Param: &internal.Param{
				Id:      0,
				Name:    paramName,
				ValType: paramValueType,
				EI: &internal.EI{
					Id:        0,
					Name:      paramEiName,
					ShortName: paramEiShortName,
				},
			},
			Value: value,
		})
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return p, nil
}

func (do *DbOperator) r_ClassProducts(tx pgx.Tx, idClass int) ([]*internal.Product, error) {
	var pp []*internal.Product
	rows, err := tx.Query(context.Background(),
		`SELECT P.ID_PRODUCT 
			FROM PRODUCTS P JOIN CLASSES C ON P.ID_PARENT_CLASS = C.ID_CLASS
			WHERE C.ID_CLASS = $1`,
		idClass)
	if err != nil {
		return nil, err
	}
	var ids []int
	var id int
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		p, err := do.r_Product(tx, id)
		if err != nil {
			return nil, err
		}
		pp = append(pp, p)
	}
	return pp, nil
}

func (do *DbOperator) u_Product(tx pgx.Tx, p *internal.Product) (err error) {
	if err = do.d_Product(tx, p.Id); err != nil {
		return
	}
	return do.c_Product(tx, p)
}

func (do *DbOperator) d_Product(tx pgx.Tx, id int) (err error) {
	var idCheck int
	if err = tx.QueryRow(context.Background(),
		`SELECT ID_PRODUCT
			FROM PRODUCTS
			WHERE ID_PRODUCT = $1`,
			id).Scan(&idCheck); err != nil {
				return err
	}
	_, err = tx.Exec(context.Background(),
		`DELETE FROM PRODUCTS 
			WHERE ID_PRODUCT = $1`,
		id)
	return
}

func (do *DbOperator) CreateProducts(pp []*internal.Product) (err error) {
	f := func(tx pgx.Tx) error {
		for _, p := range pp {
			if err := do.c_Product(tx, p); err != nil {
				return err
			}
		}
		return nil
	}
	return do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) ReadProduct(id int) (*internal.Product, error) {
	var p *internal.Product
	f := func(tx pgx.Tx) error {
		pr, err := do.r_Product(tx, id)
		if err != nil {
			return err
		}
		p = pr
		return nil
	}
	return p, do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) ReadClassProducts(id int) ([]*internal.Product, error) {
	var pp []*internal.Product
	f := func(tx pgx.Tx) error {
		products, err := do.r_ClassProducts(tx, id)
		if err != nil {
			return err
		}
		pp = products
		return nil
	}
	return pp, do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) UpdateProduct(p *internal.Product) error {
	f := func(tx pgx.Tx) error {
		if err := do.u_Product(tx, p); err != nil {
			return err
		}
		return nil
	}
	return do.cs.WrapIntoTransaction(context.Background(), f)
}

func (do *DbOperator) DeleteProduct(id int) error {
	f := func(tx pgx.Tx) error {
		if err := do.d_Product(tx, id); err != nil {
			return err
		}
		return nil
	}
	return do.cs.WrapIntoTransaction(context.Background(), f)
}

// PRODUCT PARAMS

func (do *DbOperator) c_ProductParams(tx pgx.Tx, p *internal.Product) (err error) {
	idProduct, err := do.r_ProductId(tx, p.Name)
	if err != nil {
		return err
	}
	classId, err := do.r_ClassId(tx, p.ParentClass.Name)
	if err != nil {
		return err
	}
	class, err := do.r_Class(tx, classId, true)
	if err != nil {
		return err
	}
	classParams := make(map[string]*internal.Param)
	for _, p := range class.Params {
		classParams[p.Name] = p
	}
	for _, pnv := range p.Params {
		var idParam int
		searchedP, ok := classParams[pnv.Param.Name]
		if !ok {
			return errors.New("couldn't find param")
		}
		if err = tx.QueryRow(context.Background(),
			`SELECT ID_CLASS_PARAM 
				FROM CLASS_PARAMS
				WHERE ID_CLASS = $1 AND ID_PARAM = $2`,
			searchedP.IdParamOwner, searchedP.Id).Scan(&idParam); err != nil {
			return
		}
		_, err = tx.Exec(context.Background(),
			`INSERT INTO PRODUCT_PARAM_VALUES(ID_PRODUCT, ID_PARAM, VALUE)
				VALUES ($1,$2,$3)`,
			idProduct, idParam, pnv.Value)
		if err != nil {
			return
		}
	}
	return nil
}
