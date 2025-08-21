package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

// LoginHandler ...
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	ingressPath := r.Header.Get("X-Ingress-Path")

	// log.Println(r.Method, "/login", ingressPath)

	w.Header().Set("Content-Type", "text/html")

	var loginError string

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			log.Printf("ParseForm() err: %v", err)
			loginError = "parse form error"
		} else {
			phone := r.FormValue("phone")
			// use new API layer if possible
			accounts, err := h.API.Accounts(r.Context(), phone)
			if err != nil {
				loginError = fmt.Sprintf("login error: %v", err.Error())
			} else {
				if n, err := strconv.Atoi(phone); err == nil {
					h.Config.Login = n

					if err = h.Config.WriteConfig(); err != nil {
						log.Println("error on write config file ", err)
					}
				}

				h.UserAccounts = accounts
				// log.Printf("got accounts %+v\n", accounts)

				data := AccountsPageData{accounts, phone, ingressPath, loginError}

				var tmpl []byte
				var err error
				if tmpl, err = h.TemplateFs.ReadFile("templates/accounts.html"); err != nil {
					fmt.Println(err)
				}

				t := template.New("t")
				t, err = t.Parse(string(tmpl))
				if err != nil {
					loginError = err.Error()
				} else {
					err = t.Execute(w, data)
					if err != nil {
						loginError = err.Error()
					}
				}
			}

			if loginError != "" {
				log.Println(loginError)
			}
			return
		}
	}
}
func (h *Handler) LoginAddressHandler(w http.ResponseWriter, r *http.Request) { // still transitional, uses API wrapper
	ingressPath := r.Header.Get("X-Ingress-Path")

	// log.Println(r.Method, "/login/address", ingressPath)

	w.Header().Set("Content-Type", "text/html")

	var loginError, phone, index string

	if err := r.ParseForm(); err != nil {
		loginError = fmt.Sprintf("ParseForm() err: %v", err)
	} else {
		phone = r.FormValue("phone")
		index = r.FormValue("index")

		if accountIndex, err := strconv.Atoi(index); err != nil {
			loginError = fmt.Sprintf("ParseForm() err: %v", err)
		} else {
			if accountIndex < 0 || accountIndex > len(h.UserAccounts)-1 {
				loginError = "Selected wrong account"
			} else {
				account := h.UserAccounts[accountIndex]
				h.Account = &account
				err := h.API.RequestCode(r.Context(), phone, account)
				if err != nil {
					loginError = fmt.Sprintf("loginAddress error: %v", err.Error())
				}
				if n, err := strconv.Atoi(phone); err == nil {
					h.Config.Login = n
				}
				h.Config.Operator = int(h.Account.OperatorID)
				if err = h.Config.WriteConfig(); err != nil {
					log.Println("error on write config file ", err)
				}

			}
		}

	}

	if loginError != "" {
		log.Println(loginError)
	}

	data := SMSPageData{phone, index, ingressPath, loginError}

	var tmpl []byte
	var err error
	if tmpl, err = h.TemplateFs.ReadFile("templates/sms.html"); err != nil {
		fmt.Println(err)
	}

	t := template.New("t")
	t, err = t.Parse(string(tmpl))
	if err != nil {
		loginError = err.Error()
	} else {
		err = t.Execute(w, data)
		if err != nil {
			loginError = err.Error()
		}
	}

	if loginError != "" {
		log.Println(loginError)
	}
}

// HomeHandler ...
func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	ingressPath := r.Header.Get("X-Ingress-Path")
	// log.Println(r.Method, "/", ingressPath)

	if h.Config.Token == "" || h.Config.RefreshToken == "" {
		http.Redirect(w, r, ingressPath+"/login", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	var loginError string

	hostIP, err2 := h.HANetwork()
	if err2 != nil {
		// loginError = "hostIP got: " + err2.Error()
		hostIP = "localhost"
	}

	var cameras Cameras
	var places Places
	var finances Finances
	if h.API != nil && loginError == "" {
		if cams, err := h.API.Cameras(r.Context()); err != nil {
			loginError = "cameras api error: " + err.Error()
		} else {
			cameras = cams
		}
	}
	if h.API != nil && loginError == "" {
		if pls, err := h.API.Places(r.Context()); err != nil {
			loginError = "places api error: " + err.Error()
		} else {
			places = pls
		}
	}
	if h.API != nil && loginError == "" {
		if fin, err := h.API.Finances(r.Context()); err != nil {
			loginError = "finances api error: " + err.Error()
		} else {
			finances = fin
		}
	}

	data := HomePageData{
		HassioIngress: ingressPath,
		HostIP:        hostIP,
		Port:          strconv.Itoa(h.Config.Port),
		LoginError:    loginError,
		Phone:         strconv.Itoa(h.Config.Login),
		Token:         h.Config.Token,
		RefreshToken:  h.Config.RefreshToken,
		Cameras:       cameras,
		Places:        places,
		Finances:      finances,
	}

	var tmpl []byte
	var err error
	if tmpl, err = h.TemplateFs.ReadFile("templates/home.html"); err != nil {
		fmt.Println("reafile templates/home.html error", err)
	}

	t := template.New("t")
	t, err = t.Parse(string(tmpl))
	if err != nil {
		loginError = "parse templates/home.html " + err.Error()
	} else {
		err = t.Execute(w, data)
		if err != nil {
			loginError = "execute templates/home.html " + err.Error()
		}
	}

	if loginError != "" {
		log.Println(loginError)
	}
}

// AccountsHandler ...
func (h *Handler) AccountsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, err := h.API.Accounts(r.Context(), strconv.Itoa(h.Config.Login))
	if err != nil {
		writeJSONError(w, err)
		return
	}
	if b, e := json.Marshal(data); e == nil {
		w.Write(b)
	} else {
		writeJSONError(w, e)
	}
}

// LoginSMSHandler ...
func (h *Handler) LoginSMSHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	access, refresh, err := h.API.ConfirmCode(r.Context(), strconv.Itoa(h.Config.Login), r.FormValue("code"), *h.Account)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	h.Config.Token = access
	h.Config.RefreshToken = refresh
	_ = h.Config.WriteConfig()
	b, _ := json.Marshal(map[string]string{"accessToken": access, "refreshToken": refresh})
	w.Write(b)
}
