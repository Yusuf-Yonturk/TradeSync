package main

import "errors"

// Gateway disinda biri de topic'e yazabilir, engine kendi girdisini dogrular.
func validateIncoming(in IncomingOrder) error {
	switch {
	case in.ID == "":
		return errors.New("id bos")
	case in.Symbol == "":
		return errors.New("symbol bos")
	case in.Side != "BUY" && in.Side != "SELL":
		return errors.New("gecersiz side")
	case in.Price <= 0:
		return errors.New("price sifirdan buyuk olmali")
	case in.Quantity <= 0:
		return errors.New("quantity sifirdan buyuk olmali")
	}
	return nil
}
