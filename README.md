# **URL Shortening Service (Go)**
![Coverage](https://img.shields.io/badge/Coverage-59.6%25-yellow)

Create short URLs, resolve shortened URLs, and fetch all shortened URLs

- [Development](#development)
- [Short URL API](#api)
  - [Base Config]
  - [Resolve URL]
  - API: [Create URL] | [Get URL] | [Get all URLs] | [Update URL] | [Delete URL]

## **Development**

- Create `.env` file from `.env.example` and populate with correct development variables
- `go run cmd`

Loosely based on Nic Jackson's [microservice tutorials](https://github.com/nicholasjackson/building-microservices-youtube/tree/episode_4)

### Packages:
#### shorty
- Business logic for `Link`s
- Contains `Store` interface to be implemented in data access layer(s).
```go
ShortyStore interface {
	CreateLink(ctx context.Context, newLink Link) (Link, error)
	GetLink(ctx context.Context, code string) (Link, error)
	GetLinks(ctx context.Context) (Links, error)
	UpdateLink(ctx context.Context, code string) (Link, error)
	DeleteLink(ctx context.Context, code string) (int, error)
}
```

#### handlers
- HTTP route handlers that call the CRUD methods on the given `store` implementation.
```go
func (s *ShortyService) createLink(w http.ResponseWriter, r *http.Request) {
        // Parse JSON body into a link
        linkInput.FromJSON(r.Body)
       
        // Error handling omitted for brevity... 

	// Create and save the short link to the DB
	newLink, err := s.store.CreateLink(r.Context(), linkInput)
	
	// Send new link JSON
	if err = newLink.ToJSON(w); err != nil {
		http.Error(w, "Problem marshaling your short link", http.StatusInternalServerError)
	}
}

```

#### mongodb
- Data access layer,  implemented for MongoDB

#### inmem
- Data access layer implemented for an in-memory store for simpler API testing
```go
func (i *Store) CreateLink(ctx context.Context, newLink shorty.Link) (shorty.Link, error) {
	newLink.GenCode(i.BaseURL())
	i.store[newLink.Code] = newLink
	return newLink, nil
}
```

#### function
- Entrypoint in to the Cloud function 
- All packages tied together here:
```go
func init() {
	functions.HTTP("ServeShorty", NewMux().ServeHTTP)
}

var store shorty.ShortyStore

func NewMux() *http.ServeMux {
	store, err := initStore()
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/urls", handlers.NewService(store).ServeHTTP)
	mux.HandleFunc("/", handlers.Resolver)

	return mux
}
```


#### **Test**

- `go test`

---

# API

## **Base config**

API `key` is **_required_** on all requests, **not including `GET /:code` to [resolve the short URL](#resolve-short-url)**.

- Base URL: `https://ospk.org`
- Headers: { "key": "API_KEY" }

## **Resolve short URL**

```
GET /:code
Response: 301 permanent redirect
```

## **Create short URL** _(authenticated)_

```
POST      /api/urls
Headers:   key=$API_KEY
Body:     { "originalUrl": "https://..." }
Response: "https://ospk.org/:code"
```

**Request body properties**

| Key        | Type     | Required | Description                          |
| ---------- | -------- | -------- | ------------------------------------ |
| url        | `string` | `true`   | Original URL                         |
| customCode | `string` |          | Custom endpoint - Defaults to `code` |
| createdBy  | `string` |          | User or bot that created the link    |

---

## **Fetch URL** _(authenticated)_

Fetch short URL object

```
GET /api/urls/:code
Headers:   key=$API_KEY
```

**Example Response:**

- See [Short URL Properties] for more details

```json
{
  "code": "a1b2c3d4e5",
  "customCode": "a1b2c3d4e5",
  "shortUrl": "https://ospk.org/a1b2c3d4e5",
  "originalUrl": "https://oparationspark.org/infoSession",
  "totalClicks": 0,
  "createdBy": "user name",
  "createdAt": "2022-10-21T03:17:15.400Z",
  "updatedAt": "2022-10-21T03:17:15.400Z"
}
```

---

## **Fetch all URLs** _(authenticated)_

```
GET /api/urls
Headers:   key=$API_KEY
```

**Example Response:**

- See [Short URL Properties] for more details

```json
[
  {"..."},

  {
    "code": "a1b2c3d4e5",
    "customCode": "signup",
    "shortUrl": "https://ospk.org/signup",
    "originalUrl": "https://oparationspark.org/infoSession",
    "totalClicks": 0,
    "createdBy": "user name",
    "createdAt": "2022-10-21T03:17:15.400Z",
    "updatedAt": "2022-10-21T03:17:15.400Z"
  },

  {"..."}
]
```

## **Update URL** _(authenticated)_

```
PUT /api/urls/:code
```

**Request body properties**

| Key        | Type     | Description                          |
| ---------- | -------- | ------------------------------------ |
| url        | `string` | Original URL                         |
| customCode | `string` | Custom endpoint - Defaults to `code` |
| createdBy  | `string` | User or bot that created the link    |

**Example Request Body:**

- See [Short URL Properties] for more details

```json
{
  "customCode": "info",
  "originalUrl": "https://oparationspark.org/info-session",
  "createdBy": "User Name"
}
```

**Example Response:**

- See [Short URL Properties] for more details

```json
{
  "code": "a1b2c3d4e5",
  "customCode": "a1b2c3d4e5",
  "shortUrl": "https://ospk.org/a1b2c3d4e5",
  "originalUrl": "https://oparationspark.org/info-session",
  "totalClicks": 0,
  "createdBy": "User Name",
  "createdAt": "2022-10-21T03:17:15.400Z",
  "updatedAt": "2022-10-21T03:17:15.400Z"
}
```

## **Delete URL** _(authenticated)_

```
DELETE /api/urls/:code
Headers:   key=$API_KEY
Response Status: 200 | 404
```

### Short URL Properties

| Key         | Type     | Edit   | Description                          |
| ----------- | -------- | ------ | ------------------------------------ |
| code        | `string` |        | randomized 10 character code         |
| customCode  | `string` | `true` | custom endpoint - Defaults to `code` |
| shortUrl    | `string` |        | short url                            |
| originalUrl | `string` | `true` | Full URL originally provided         |
| createdBy   | `string` | `true` | User or bot that created the link    |
| totalClicks | `number` |        | Total clicks (Allows duplicates)     |
| createdAt   | `Date`   |        | Date created                         |
| updatedAt   | `Date`   |        | Date last modified                   |

[short url properties]: #short-url-properties
[base config]: #base-config
[resolve url]: #resolve-short-url
[create url]: #create-short-url-authenticated
[get url]: #fetch-url-authenticated
[get all urls]: #fetch-all-urls-authenticated
[update url]: #update-url-authenticated
[delete url]: #delete-url-authenticated
