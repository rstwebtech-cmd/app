package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"sort"
	"strings"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"encoding/base64"
	"log"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"sync"
	"time"
)

type User struct {
	ID, Name, Phone, Email, Level, Roll string
	Courses                []string
}
type Course struct {
	ID, Title, Category, Duration, Image string
	Price, MRP                    int
	OfferLabel                    string // e.g. "Diwali Sale"
	OfferEnds                     string // RFC3339, app shows countdown
	Lessons []Lesson
}
type Combo struct {
	ID, Title string
	Items     []string
	MRP, Price int
}
type Review struct {
	ID, CourseID, Name, Text string
	Stars                    int
	Approved                 bool
}
type Group struct{ ID, Name, Owner string }
type Msg struct {
	ID            int
	Room, Name, Text string
	At            int64
}
type Q struct {
	Q       string
	Options []string
	Answer  int
}
type Order struct {
	ID, UserID, Item string
	Amount           int
	Paid             bool
	At               int64
	Coupon           string
}
type Live struct{ ID, Title, At, Link string }
type Lesson struct {
	Title, URL string
	Free       bool
}
type Coupon struct {
	Code                           string
	Percent, Flat, MaxUses, Used   int
	Expires                        string
	Active                         bool
}
type File struct {
	Type string
	Data []byte
}
type Cert struct{ ID, UserName, CourseTitle, Date, Roll string }
type DB struct {
	Users    map[string]*User
	Courses  []Course
	Combos   []Combo
	Reviews  []Review
	Groups   []Group
	Msgs     []Msg
	Quizzes  map[string][]Q
	Certs    map[string]Cert
	Orders   map[string]*Order
	Lives    []Live
	Coupons  []Coupon
	Creds    map[string]string
	Sessions map[string]string
	Files    map[string]File `json:"-"`
	Settings map[string]string // smtp_host,smtp_port,smtp_user,smtp_pass,wa_url,wa_token
}

var (
	db       = &DB{Users: map[string]*User{}, Quizzes: map[string][]Q{}, Certs: map[string]Cert{}, Orders: map[string]*Order{}, Creds: map[string]string{}, Files: map[string]File{}, Sessions: sessions, Settings: map[string]string{}}
	mu       sync.Mutex
	sessions = map[string]string{}
	otps     = map[string]string{}
)

//go:embed static
var staticFS embed.FS

//go:embed checkout.html
var checkoutHTML []byte

//go:embed embed.js
var embedJS []byte

//go:embed admin.html
var adminHTML []byte

func rest(cmd ...any) json.RawMessage {
	u, t := os.Getenv("STORE_URL"), os.Getenv("STORE_TOKEN")
	if u == "" {
		return nil
	}
	b, _ := json.Marshal(cmd)
	req, _ := http.NewRequest("POST", u, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+t)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var o struct{ Result json.RawMessage }
	json.NewDecoder(resp.Body).Decode(&o)
	return o.Result
}
func save() {
	b, _ := json.Marshal(db)
	os.WriteFile("data.json", b, 0644)
	rest("SET", "jp:data", string(b))
}
func fileGet(id string) (File, bool) {
	if f, ok := db.Files[id]; ok {
		return f, true
	}
	var sv string
	if json.Unmarshal(rest("GET", "jp:f:"+id), &sv) == nil && sv != "" {
		if i := strings.Index(sv, "|"); i > 0 {
			if d, err := base64.StdEncoding.DecodeString(sv[i+1:]); err == nil {
				f := File{sv[:i], d}
				db.Files[id] = f
				return f, true
			}
		}
	}
	return File{}, false
}
func hashPw(salt, pw string) string {
	h := sha256.Sum256([]byte(salt + pw))
	return salt + "$" + hex.EncodeToString(h[:])
}
func userByEmail(e string) *User {
	for _, u := range db.Users {
		if strings.EqualFold(u.Email, e) {
			return u
		}
	}
	return nil
}
func price(item, code string) (int, string) {
	amt := 0
	for _, c := range db.Courses {
		if c.ID == item {
			amt = c.Price
		}
	}
	for _, c := range db.Combos {
		if c.ID == item {
			amt = c.Price
		}
	}
	if amt > 0 && code != "" {
		for i := range db.Coupons {
			c := &db.Coupons[i]
			if strings.EqualFold(c.Code, code) && c.Active && (c.MaxUses == 0 || c.Used < c.MaxUses) && (c.Expires == "" || c.Expires >= time.Now().Format("2006-01-02")) {
				d := amt*c.Percent/100 + c.Flat
				if d > amt-1 {
					d = amt - 1
				}
				return amt - d, c.Code
			}
		}
	}
	return amt, ""
}
func pubCourses() []Course {
	out := make([]Course, len(db.Courses))
	copy(out, db.Courses)
	for i := range out {
		out[i].Lessons = nil
	}
	return out
}
func fix() {
	if db.Users == nil {
		db.Users = map[string]*User{}
	}
	if db.Quizzes == nil {
		db.Quizzes = map[string][]Q{}
	}
	if db.Certs == nil {
		db.Certs = map[string]Cert{}
	}
	if db.Settings == nil {
		db.Settings = map[string]string{}
	}
	if db.Orders == nil {
		db.Orders = map[string]*Order{}
	}
	if db.Creds == nil {
		db.Creds = map[string]string{}
	}
	if db.Files == nil {
		db.Files = map[string]File{}
	}
	if db.Sessions == nil {
		db.Sessions = map[string]string{}
	}
	sessions = db.Sessions
}
func load() {
	var raw []byte
	var sv string
	if json.Unmarshal(rest("GET", "jp:data"), &sv) == nil && sv != "" {
		raw = []byte(sv)
	} else if b, err := os.ReadFile("data.json"); err == nil {
		raw = b
	}
	if len(raw) > 0 {
		json.Unmarshal(raw, db)
		fix()
		return
	}
	for i, c := range []Course{
		{Title: "Basic English Language Course", Category: "English", Duration: "941m", Price: 2859, MRP: 3999, Image: "basic.jpg"},
		{Title: "Advance English Course", Category: "English", Duration: "472m", Price: 3449, MRP: 4499, Image: "advance.jpg"},
		{Title: "A1-A2 English Course", Category: "English", Duration: "500m", Price: 2999, MRP: 3599, Image: "a1a2.jpg"},
		{Title: "B1-B2 English Course", Category: "English", Duration: "500m", Price: 2999, MRP: 3599, Image: "b1b2.jpg"},
		{Title: "British Accent Course", Category: "Accent", Duration: "241m", Price: 4039, MRP: 8999, Image: "british.jpg"},
		{Title: "American Accent Spoken Course", Category: "Accent", Duration: "332m", Price: 4039, MRP: 6999, Image: "american.jpg"},
		{Title: "French Spoken Course", Category: "Languages", Duration: "291m", Price: 2859, MRP: 3999, Image: "french.jpg"},
		{Title: "Spanish Full Course", Category: "Languages", Duration: "209m", Price: 2559, MRP: 3999, Image: "spanish.jpg"},
		{Title: "Japanese Language Course", Category: "Languages", Duration: "329m", Price: 2859, MRP: 3999, Image: "japanese.jpg"},
		{Title: "IELTS Full Course", Category: "Career", Duration: "335m", Price: 4039, MRP: 8999, Image: "ielts.jpg"},
		{Title: "Interview Classes - Hindi", Category: "Career", Duration: "188m", Price: 3539, MRP: 4999, Image: "interview_hindi.jpg"},
		{Title: "Interview Classes - English", Category: "Career", Duration: "189m", Price: 3539, MRP: 4999},
	} {
		c.ID = fmt.Sprint("c", i+1)
		if c.Image != "" {
			c.Image = "/static/" + c.Image
		}
		c.OfferLabel = "Limited Offer"
		c.OfferEnds = time.Now().Add(72 * time.Hour).Format(time.RFC3339)
		db.Courses = append(db.Courses, c)
		db.Quizzes[c.ID] = []Q{{"Which is a greeting?", []string{"Hello", "Table", "Run"}, 0}, {"Past of 'go'?", []string{"goed", "went", "gone"}, 1}, {"Plural of 'child'?", []string{"childs", "children", "childes"}, 1}}
	}
	db.Combos = []Combo{{"b1", "Spoken English + Accent Training", []string{"c1", "c5"}, 29999, 17999}}
	db.Groups = []Group{{"g1", "Janhavi Community", "admin"}}
	db.Quizzes["level"] = []Q{{"I ___ a student.", []string{"am", "is", "are"}, 0}, {"She ___ to school every day.", []string{"go", "goes", "going"}, 1}, {"They ___ playing now.", []string{"is", "are", "was"}, 1}, {"I have lived here ___ 2010.", []string{"for", "since", "from"}, 1}, {"If it rains, we ___ stay home.", []string{"will", "would", "did"}, 0}, {"He asked me where I ___.", []string{"live", "lived", "living"}, 1}, {"The report ___ by tomorrow.", []string{"will be finished", "finishes", "is finishing"}, 0}, {"Had I known, I ___ come.", []string{"would have", "will", "had"}, 0}, {"She is ___ than her sister.", []string{"tall", "taller", "tallest"}, 1}, {"Hardly ___ arrived when it began to rain.", []string{"had he", "he had", "did he"}, 0}}
}

func rid() string { b := make([]byte, 8); rand.Read(b); return hex.EncodeToString(b) }
func jw(w http.ResponseWriter, v any) { w.Header().Set("Content-Type", "application/json"); json.NewEncoder(w).Encode(v) }
func in(r *http.Request, v any)       { json.NewDecoder(r.Body).Decode(v) }
func me(r *http.Request) *User {
	t := r.Header.Get("Authorization")
	if len(t) > 7 {
		return db.Users[sessions[t[7:]]]
	}
	return nil
}
func guard(admin bool, f func(http.ResponseWriter, *http.Request, *User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		defer func() {
			if r.Method != "GET" {
				save()
			}
		}()
		if admin && r.Header.Get("X-Admin-Key") != os.Getenv("ADMIN_KEY") {
			http.Error(w, `{"error":"admin only"}`, 403)
			return
		}
		u := me(r)
		if !admin && u == nil && r.URL.Path != "/api/courses" && r.URL.Path != "/api/combos" && r.URL.Path != "/api/live" && r.URL.Path != "/api/reviews" && r.URL.Path != "/api/auth/otp/send" && r.URL.Path != "/api/auth/otp/verify" && r.URL.Path != "/api/auth/google" && r.URL.Path != "/api/auth/register" && r.URL.Path != "/api/auth/login" {
			http.Error(w, `{"error":"login required"}`, 401)
			return
		}
		f(w, r, u)
	}
}
func login(w http.ResponseWriter, u *User) {
	t := rid()
	sessions[t] = u.ID
	jw(w, map[string]any{"token": t, "user": u})
}
func findUser(phone, email, name string) *User {
	for _, u := range db.Users {
		if (phone != "" && u.Phone == phone) || (email != "" && u.Email == email) {
			return u
		}
	}
	u := &User{ID: rid(), Name: name, Phone: phone, Email: email, Roll: fmt.Sprintf("JP%05d", len(db.Users)+1)}
	db.Users[u.ID] = u
	return u
}
func sendEmail(to, sub, body string) {
	s := db.Settings
	if s["smtp_host"] == "" {
		return
	}
	msg := "Subject: " + sub + "\r\n\r\n" + body
	smtp.SendMail(s["smtp_host"]+":"+s["smtp_port"], smtp.PlainAuth("", s["smtp_user"], s["smtp_pass"], s["smtp_host"]), s["smtp_user"], []string{to}, []byte(msg))
}
func sendWA(to, body string) {
	s := db.Settings
	if s["wa_url"] == "" {
		return
	}
	b, _ := json.Marshal(map[string]string{"to": to, "message": body})
	req, _ := http.NewRequest("POST", s["wa_url"], bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+s["wa_token"])
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)
}

func main() {
	load()
	m := http.NewServeMux()
	// ---- auth ----
	m.HandleFunc("POST /api/auth/otp/send", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var b struct{ Phone string }
		in(r, &b)
		n, _ := rand.Int(rand.Reader, big.NewInt(900000))
		code := fmt.Sprint(100000 + n.Int64())
		otps[b.Phone] = code
		go sendWA(b.Phone, "Your Janhavi Panwar login OTP: "+code) // or plug an SMS provider here
		log.Println("OTP", b.Phone, code)                          // DEV: see console
		jw(w, map[string]string{"status": "sent"})
	}))
	m.HandleFunc("POST /api/auth/otp/verify", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var b struct{ Phone, Code, Name string }
		in(r, &b)
		if otps[b.Phone] == "" || otps[b.Phone] != b.Code {
			http.Error(w, `{"error":"wrong otp"}`, 400)
			return
		}
		delete(otps, b.Phone)
		login(w, findUser(b.Phone, "", b.Name))
	}))
	m.HandleFunc("POST /api/auth/google", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var b struct{ IDToken string }
		in(r, &b)
		resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + b.IDToken)
		var g struct{ Email, Name, Aud string }
		if err == nil && resp.StatusCode == 200 {
			json.NewDecoder(resp.Body).Decode(&g)
		}
		if g.Email == "" || (os.Getenv("GOOGLE_CLIENT_ID") != "" && g.Aud != os.Getenv("GOOGLE_CLIENT_ID")) {
			http.Error(w, `{"error":"invalid google token"}`, 401)
			return
		}
		login(w, findUser("", g.Email, g.Name))
	}))
	// ---- catalogue ----
	m.HandleFunc("GET /api/courses", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, pubCourses()) }))
	m.HandleFunc("GET /api/combos", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db.Combos) }))
	m.HandleFunc("GET /api/me", guard(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		var cs []Cert
		for _, c := range db.Certs {
			if c.UserName == u.Name {
				cs = append(cs, c)
			}
		}
		jw(w, map[string]any{"user": u, "certs": cs})
	}))
	// ---- reviews (hidden until approved) ----
	m.HandleFunc("GET /api/reviews", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) {
		out := []Review{}
		for _, v := range db.Reviews {
			if v.Approved {
				out = append(out, v)
			}
		}
		jw(w, out)
	}))
	m.HandleFunc("POST /api/reviews", guard(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		var v Review
		in(r, &v)
		v.ID, v.Name, v.Approved = rid(), u.Name, false
		db.Reviews = append(db.Reviews, v)
		jw(w, map[string]string{"status": "submitted, pending approval"})
	}))
	// ---- groups & live chat (poll /api/chat?room=..&after=id) ----
	m.HandleFunc("GET /api/groups", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db.Groups) }))
	m.HandleFunc("POST /api/groups", guard(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		var g Group
		in(r, &g)
		g.ID, g.Owner = rid(), u.ID
		db.Groups = append(db.Groups, g)
		jw(w, g)
	}))
	m.HandleFunc("GET /api/chat", guard(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		if rm := r.URL.Query().Get("room"); strings.HasPrefix(rm, "support-") && rm != "support-"+u.ID {
			http.Error(w, `{"error":"not allowed"}`, 403)
			return
		}
		after := 0
		fmt.Sscan(r.URL.Query().Get("after"), &after)
		out := []Msg{}
		for _, x := range db.Msgs {
			if x.Room == r.URL.Query().Get("room") && x.ID > after {
				out = append(out, x)
			}
		}
		jw(w, out)
	}))
	m.HandleFunc("POST /api/chat", guard(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		var x Msg
		in(r, &x)
		if strings.HasPrefix(x.Room, "support-") && x.Room != "support-"+u.ID {
			http.Error(w, `{"error":"not allowed"}`, 403)
			return
		}
		x.ID, x.Name, x.At = len(db.Msgs)+1, u.Name, time.Now().Unix()
		db.Msgs = append(db.Msgs, x)
		jw(w, x)
	}))
	// ---- quiz + certificate ----
	m.HandleFunc("GET /api/quiz/{id}", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) {
		jw(w, db.Quizzes[r.PathValue("id")]) // NOTE: app strips nothing; hide answers before production
	}))
	m.HandleFunc("POST /api/quiz/{id}", guard(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		var b struct{ Answers []int }
		in(r, &b)
		qs, score := db.Quizzes[r.PathValue("id")], 0
		for i, q := range qs {
			if i < len(b.Answers) && b.Answers[i] == q.Answer {
				score++
			}
		}
		pass := len(qs) > 0 && score*100/len(qs) >= 60
		res := map[string]any{"score": score, "total": len(qs), "passed": pass}
		if r.PathValue("id") == "level" {
			lv := []string{"A1", "A1", "A2", "A2", "B1", "B1", "B2", "B2", "C1", "C1", "C1"}
			i := score
			if i >= len(lv) {
				i = len(lv) - 1
			}
			u.Level = lv[i]
			res["level"] = u.Level
			jw(w, res)
			return
		}
		if pass {
			title := ""
			for _, c := range db.Courses {
				if c.ID == r.PathValue("id") {
					title = c.Title
				}
			}
			c := Cert{rid(), u.Name, title, time.Now().Format("02 Jan 2006"), fmt.Sprintf("JP-%d-%04d", time.Now().Year(), len(db.Certs)+1)}
			db.Certs[c.ID] = c
			res["certificate"] = c.ID
		}
		jw(w, res)
	}))
	m.HandleFunc("GET /certificate/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		c, ok := db.Certs[r.PathValue("id")]
		mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		signer := db.Settings["cert_signer"]
		if signer == "" {
			signer = "Janhavi Panwar"
		}
		fmt.Fprintf(w, `<meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1"><style>body{margin:0;background:#f5f3ff;font-family:Georgia,serif}.c{max-width:820px;margin:20px auto;background:#fff;border:14px double #6d28d9;padding:40px;text-align:center}h1{color:#2e1065;letter-spacing:3px;margin:8px 0}h2{color:#6d28d9;font-size:34px;margin:10px 0}.g{color:#b45309}.r{display:flex;justify-content:space-between;margin-top:40px;font-size:14px;color:#444}@media print{button{display:none}body{background:#fff}}</style><div class=c><img src="/static/logo.png" height=50><h1>CERTIFICATE OF COMPLETION</h1><p>This is to certify that</p><h2>%s</h2><p>has successfully completed the course</p><h2 class=g>%s</h2><div class=r><span>Roll No: <b>%s</b></span><span>Date: %s</span><span><b>%s</b><br>Wonder Girl of India</span></div><p style="font-size:11px;color:#999">Certificate ID: %s</p></div><center><button onclick=print()>Download / Print</button></center>`, html.EscapeString(c.UserName), html.EscapeString(c.CourseTitle), c.Roll, c.Date, html.EscapeString(signer), c.ID)
	})
	// ---- ADMIN (header X-Admin-Key) ----
	m.HandleFunc("GET /api/admin/reviews", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db.Reviews) }))
	m.HandleFunc("POST /api/admin/reviews/{id}/approve", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		for i := range db.Reviews {
			if db.Reviews[i].ID == r.PathValue("id") {
				db.Reviews[i].Approved = true
			}
		}
		jw(w, "ok")
	}))
	m.HandleFunc("POST /api/admin/courses", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) { // add/update course & offer
		var c Course
		in(r, &c)
		if c.ID == "" {
			c.ID = rid()
			db.Courses = append(db.Courses, c)
		} else {
			for i := range db.Courses {
				if db.Courses[i].ID == c.ID {
					db.Courses[i] = c
				}
			}
		}
		jw(w, c)
	}))
	m.HandleFunc("POST /api/admin/combos", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var c Combo
		in(r, &c)
		c.ID = rid()
		db.Combos = append(db.Combos, c)
		jw(w, c)
	}))
	m.HandleFunc("POST /api/admin/quiz/{id}", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var qs []Q
		in(r, &qs)
		db.Quizzes[r.PathValue("id")] = qs
		jw(w, "ok")
	}))
	m.HandleFunc("GET /api/admin/students", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db.Users) }))
	m.HandleFunc("PUT /api/admin/settings", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) { // API keys / SMTP / WhatsApp
		var s map[string]string
		in(r, &s)
		for k, v := range s {
			db.Settings[k] = v
		}
		jw(w, "saved")
	}))
	m.HandleFunc("POST /api/admin/broadcast", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var b struct {
			Channel, Subject, Message string // channel: email | whatsapp
			UserIDs                   []string // empty = all students
		}
		in(r, &b)
		n := 0
		for id, u := range db.Users {
			if len(b.UserIDs) > 0 && !contains(b.UserIDs, id) {
				continue
			}
			if b.Channel == "email" && u.Email != "" {
				go sendEmail(u.Email, b.Subject, b.Message)
				n++
			} else if b.Channel == "whatsapp" && u.Phone != "" {
				go sendWA(u.Phone, b.Message)
				n++
			}
		}
		jw(w, map[string]int{"queued": n})
	}))
	// ---- Razorpay ----
	m.HandleFunc("POST /api/pay/order", guard(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		var b struct{ ItemID, Coupon string }
		in(r, &b)
		amt, cc := price(b.ItemID, b.Coupon)
		st := db.Settings
		if amt == 0 || st["rz_key_id"] == "" {
			http.Error(w, `{"error":"payment not configured in admin settings"}`, 400)
			return
		}
		body, _ := json.Marshal(map[string]any{"amount": amt * 100, "currency": "INR", "receipt": rid()})
		req, _ := http.NewRequest("POST", "https://api.razorpay.com/v1/orders", bytes.NewReader(body))
		req.SetBasicAuth(st["rz_key_id"], st["rz_key_secret"])
		req.Header.Set("Content-Type", "application/json")
		var o struct{ ID string }
		if resp, err := http.DefaultClient.Do(req); err == nil {
			json.NewDecoder(resp.Body).Decode(&o)
		}
		if o.ID == "" {
			http.Error(w, `{"error":"razorpay order failed - check keys"}`, 502)
			return
		}
		db.Orders[o.ID] = &Order{o.ID, u.ID, b.ItemID, amt, false, time.Now().Unix(), cc}
		sch := "https"
		if strings.HasPrefix(r.Host, "localhost") {
			sch = "http"
		}
		jw(w, map[string]string{"url": sch + "://" + r.Host + "/pay/" + o.ID})
	}))
	m.HandleFunc("GET /pay/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		o, k := db.Orders[r.PathValue("id")], db.Settings["rz_key_id"]
		mu.Unlock()
		if o == nil {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `<meta name=viewport content="width=device-width,initial-scale=1"><body style="font-family:sans-serif;text-align:center;padding:40px"><h3>Opening payment...</h3><script src="https://checkout.razorpay.com/v1/checkout.js"></script><script>new Razorpay({key:"%s",order_id:"%s",amount:%d,currency:"INR",name:"Janhavi Panwar",handler:function(r){fetch("/api/pay/verify",{method:"POST",body:JSON.stringify(r)}).then(function(x){document.body.innerHTML=x.ok?"<h2>Payment successful ✅<br>App par wapas jaao</h2>":"<h2>Verification failed</h2>"})}}).open()</script>`, k, o.ID, o.Amount*100)
	})
	m.HandleFunc("POST /api/pay/verify", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		defer save()
		var b struct{ Razorpay_order_id, Razorpay_payment_id, Razorpay_signature string }
		in(r, &b)
		o := db.Orders[b.Razorpay_order_id]
		h := hmac.New(sha256.New, []byte(db.Settings["rz_key_secret"]))
		h.Write([]byte(b.Razorpay_order_id + "|" + b.Razorpay_payment_id))
		if o == nil || o.Paid || !hmac.Equal([]byte(hex.EncodeToString(h.Sum(nil))), []byte(b.Razorpay_signature)) {
			http.Error(w, `{"error":"invalid"}`, 400)
			return
		}
		o.Paid = true
		for i := range db.Coupons {
			if o.Coupon != "" && db.Coupons[i].Code == o.Coupon {
				db.Coupons[i].Used++
			}
		}
		items := []string{o.Item}
		for _, c := range db.Combos {
			if c.ID == o.Item {
				items = c.Items
			}
		}
		if u := db.Users[o.UserID]; u != nil {
			u.Courses = append(u.Courses, items...)
		}
		jw(w, "ok")
	})
	m.HandleFunc("GET /api/admin/orders", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db.Orders) }))
	m.Handle("GET /static/", http.FileServer(http.FS(staticFS)))
	m.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(adminHTML)
	})
	m.HandleFunc("GET /api/live", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db.Lives) }))
	m.HandleFunc("POST /api/admin/live", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var l Live
		in(r, &l)
		l.ID = rid()
		db.Lives = append(db.Lives, l)
		jw(w, l)
	}))
	m.HandleFunc("POST /api/admin/live/{id}/delete", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		for i, l := range db.Lives {
			if l.ID == r.PathValue("id") {
				db.Lives = append(db.Lives[:i], db.Lives[i+1:]...)
				break
			}
		}
		jw(w, "ok")
	}))
	m.HandleFunc("GET /api/admin/stats", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		rev, paid, pend := 0, 0, 0
		days, top, names := map[string]int{}, map[string]int{}, map[string]string{}
		for _, c := range db.Courses {
			names[c.ID] = c.Title
		}
		for _, c := range db.Combos {
			names[c.ID] = c.Title
		}
		for _, o := range db.Orders {
			if o.Paid {
				rev += o.Amount
				paid++
				days[time.Unix(o.At, 0).Format("02 Jan")] += o.Amount
				top[o.Item]++
			} else {
				pend++
			}
		}
		labels, vals := []string{}, []int{}
		for i := 13; i >= 0; i-- {
			d := time.Now().AddDate(0, 0, -i).Format("02 Jan")
			labels, vals = append(labels, d), append(vals, days[d])
		}
		type T struct {
			Title string
			Sales int
		}
		tops := []T{}
		for id, n := range top {
			tops = append(tops, T{names[id], n})
		}
		sort.Slice(tops, func(a, b int) bool { return tops[a].Sales > tops[b].Sales })
		rp := 0
		for _, v := range db.Reviews {
			if !v.Approved {
				rp++
			}
		}
		jw(w, map[string]any{"revenue": rev, "orders": paid, "customers": len(db.Users), "pending": pend, "labels": labels, "values": vals, "top": tops, "reviewsPending": rp})
	}))
	m.HandleFunc("POST /api/auth/register", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var b struct{ Name, Email, Phone, Password string }
		in(r, &b)
		e := strings.ToLower(strings.TrimSpace(b.Email))
		if e == "" || b.Name == "" || len(b.Password) < 6 {
			http.Error(w, `{"error":"naam, email aur 6+ akshar ka password zaruri hai"}`, 400)
			return
		}
		if db.Creds[e] != "" {
			http.Error(w, `{"error":"is email se account pehle se hai, login karo"}`, 400)
			return
		}
		db.Creds[e] = hashPw(rid(), b.Password)
		u := userByEmail(e)
		if u == nil {
			u = &User{ID: rid(), Email: e, Roll: fmt.Sprintf("JP%05d", len(db.Users)+1)}
			db.Users[u.ID] = u
		}
		u.Name, u.Phone = b.Name, b.Phone
		login(w, u)
	}))
	m.HandleFunc("POST /api/auth/login", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var b struct{ Email, Password string }
		in(r, &b)
		e := strings.ToLower(strings.TrimSpace(b.Email))
		st := db.Creds[e]
		p := strings.SplitN(st, "$", 2)
		u := userByEmail(e)
		if st == "" || u == nil || hashPw(p[0], b.Password) != st {
			http.Error(w, `{"error":"email ya password galat hai"}`, 401)
			return
		}
		login(w, u)
	}))
	m.HandleFunc("POST /api/coupon/validate", guard(false, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var b struct{ ItemID, Code string }
		in(r, &b)
		orig, _ := price(b.ItemID, "")
		amt, cc := price(b.ItemID, b.Code)
		jw(w, map[string]any{"original": orig, "price": amt, "valid": cc != ""})
	}))
	m.HandleFunc("GET /api/course/{id}/lessons", guard(false, func(w http.ResponseWriter, r *http.Request, u *User) {
		en := contains(u.Courses, r.PathValue("id"))
		out := []Lesson{}
		for _, c := range db.Courses {
			if c.ID == r.PathValue("id") {
				for _, l := range c.Lessons {
					if !en && !l.Free {
						l.URL = ""
					}
					out = append(out, l)
				}
			}
		}
		jw(w, out)
	}))
	m.HandleFunc("GET /files/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		f, ok := fileGet(r.PathValue("id"))
		mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", f.Type)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(f.Data)
	})
	m.HandleFunc("GET /embed.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Write(embedJS)
	})
	m.HandleFunc("GET /checkout/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(checkoutHTML)
	})
	m.HandleFunc("GET /api/admin/courses", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db.Courses) }))
	m.HandleFunc("POST /api/admin/courses/import", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var l []Course
		in(r, &l)
		for _, c := range l {
			c.ID = rid()
			db.Courses = append(db.Courses, c)
		}
		jw(w, map[string]int{"imported": len(l)})
	}))
	m.HandleFunc("POST /api/admin/upload", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		r.ParseMultipartForm(2 << 20)
		f, h, err := r.FormFile("file")
		if err != nil {
			http.Error(w, `{"error":"file nahi mili"}`, 400)
			return
		}
		defer f.Close()
		ct := h.Header.Get("Content-Type")
		d, _ := io.ReadAll(io.LimitReader(f, 700<<10))
		if !(strings.HasPrefix(ct, "image/") || ct == "application/pdf") || len(d) >= 700<<10 {
			http.Error(w, `{"error":"sirf image/pdf, 700KB se chhoti"}`, 400)
			return
		}
		id := rid()
		db.Files[id] = File{ct, d}
		rest("SET", "jp:f:"+id, ct+"|"+base64.StdEncoding.EncodeToString(d))
		jw(w, map[string]string{"url": "/files/" + id})
	}))
	m.HandleFunc("GET /api/admin/coupons", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db.Coupons) }))
	m.HandleFunc("POST /api/admin/coupons", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var c Coupon
		in(r, &c)
		for i := range db.Coupons {
			if db.Coupons[i].Code == c.Code {
				c.Used = db.Coupons[i].Used
				db.Coupons[i] = c
				jw(w, c)
				return
			}
		}
		db.Coupons = append(db.Coupons, c)
		jw(w, c)
	}))
	m.HandleFunc("POST /api/admin/coupons/{code}/delete", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		for i, c := range db.Coupons {
			if c.Code == r.PathValue("code") {
				db.Coupons = append(db.Coupons[:i], db.Coupons[i+1:]...)
				break
			}
		}
		jw(w, "ok")
	}))
	m.HandleFunc("POST /api/admin/combos/{id}/delete", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		for i, c := range db.Combos {
			if c.ID == r.PathValue("id") {
				db.Combos = append(db.Combos[:i], db.Combos[i+1:]...)
				break
			}
		}
		jw(w, "ok")
	}))
	m.HandleFunc("GET /api/admin/chat/rooms", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		type R struct {
			Room, Last, Name string
			At               int64
		}
		rooms := map[string]*R{}
		for _, x := range db.Msgs {
			if strings.HasPrefix(x.Room, "support-") {
				rr := rooms[x.Room]
				if rr == nil {
					rr = &R{Room: x.Room}
					rooms[x.Room] = rr
				}
				rr.Last, rr.At = x.Text, x.At
				if x.Name != "Janhavi Team" {
					rr.Name = x.Name
				}
			}
		}
		out := []*R{}
		for _, v := range rooms {
			out = append(out, v)
		}
		sort.Slice(out, func(a, b int) bool { return out[a].At > out[b].At })
		jw(w, out)
	}))
	m.HandleFunc("GET /api/admin/chat", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		out := []Msg{}
		for _, x := range db.Msgs {
			if x.Room == r.URL.Query().Get("room") {
				out = append(out, x)
			}
		}
		jw(w, out)
	}))
	m.HandleFunc("POST /api/admin/chat", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var x Msg
		in(r, &x)
		x.ID, x.Name, x.At = len(db.Msgs)+1, "Janhavi Team", time.Now().Unix()
		db.Msgs = append(db.Msgs, x)
		jw(w, x)
	}))
	m.HandleFunc("GET /api/admin/backup", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) { jw(w, db) }))
	m.HandleFunc("POST /api/admin/restore", guard(true, func(w http.ResponseWriter, r *http.Request, _ *User) {
		var n DB
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, `{"error":"bad file"}`, 400)
			return
		}
		db = &n
		fix()
		jw(w, "restored")
	}))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("listening :" + port)
	log.Fatal(http.ListenAndServe(":"+port, cors(m)))
}
func contains(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		if r.Method == "OPTIONS" {
			return
		}
		h.ServeHTTP(w, r)
	})
}
