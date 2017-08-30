# cboxswanapid
SWAN API Daemon for CERNBox

# Preliminary version of the spec

## Authentication

### GET /authenticate

Protected by shibboleth.

Request headers

```
Referer: https://swanXXX.cern.ch
```

Response headers

```
X-Frame-Options: ALLOW-FROM https://swanXXX.example.org
```

Accessed through an iFrame. Hence it sets the header X-Frame-Options to be possible to open it as an iFrame.

Returns a page with a script that calls parent.postMessage(...) (https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage). This call should send a token with expire date (ISO format).

Response Examples

```
<script>parent.postMessage({"authtoken":"xxxx","expire":"2017-06-20 13:00:00"}, 'https://swanXXX.cern.ch');</script>
```

## Sharing API

All API requests need a valid authtoken (provided by /authenticate) in the request header:

```

Authorization: Bearer <authtoken>
Origin: https://swanXXX.example.org
```

Missing or wrong Authorization header results in 401 Unauthorized. Missing or wrong Origin header results in 400 Bad Request.


Every API reponse has the following CORS header:

```
Access-Control-Allow-Origin: https//swanXXX.example.org
```

### OPTIONS

Each API endpoint needs to implement the method OPTIONS. This is used for CORS' cross-origin HTTP requests. When a cross-origin request is done, the browser first issues a preflight OPTIONS request, asking the 
server for permission to make the actual request.

OPTIONS request are not authenticated but they require a valid Origin header.

OPTIONS request verifies the following headers:
 * Origin - check if it comes from https://swanXXX.example.org
 * Access-Control-Request-Method - check if the method asked is valid (case insensitive)
 * Access-Control-Request-Headers - check if it only contains 'Authorization', as it is the only 
 header used in this API (case insensitive)

Anything wrong with these request headers results in 400 Bad Request response.


The reply to OPTIONS request needs the following headers:

 ```
 
 Access-Control-Allow-Origin: https://swanXXX.cern.ch
 Access-Control-Allow-Methods: GET, POST, PUT, DELETE (depending on the endpoint)
 Access-Control-Allow-Headers: Authorization
 
 ```
 
In the Allow-Methods, the list should contain all the methods allowed on that endpoint, so that the browser can cache 
this reply.

 

### GET /sharing

Returns a list of all projects that I share

Response Examples

```
200

{"sharing": [
    "Swan Projects/Project 1/",
    "Swan Projects/Project 2/"
]}
```
```
200

{"sharing": [
]}
```

### GET /shared

Returns all the projects shared with me

Response Examples

```
200
{"shared": [
    {
        "user":{"name":"Diogo C.","user":"diogo"},
        "path":"/users/d/diogo/Swan Projects/Project 1",
        "size":10240,
        "date":"2017-06-20 11:00:00"
    }
]}
```

```
200
{"sharing": [
]}
```

### GET /share

Returns the people to whom I share a Project

Query Params

project: path of the project ("Swan Projects/Project 1/")

Response Examples

```
200

{"share": [
]}
```

```
200

{"share": [
    {
        "value":{"shareType":0,"shareWith":"diogo"},
        "label":"Diogo C. (diogo)"
    },
    {
        "value":{"shareType":1,"shareWith":"Admin-something"},
        "label":"Admins (Group)"
    }
]}
```


### POST+PUT /share

Shares a project with someone or updates the sharing.

Query Params

project: path of the project ("Swan Projects/Project 1/")

Body

```
{"share":[
    {"shareType":0,"shareWith":"diogo"},
    {"shareType":1,"shareWith":"Admin-something"}
]}

```

Response Examples

```
200 Ok
```

```
400

{"error":"message"}
```

### DELETE /share

Removes the sharing from a project

Query Params

project: path of the project ("Swan Projects/Project 1/")

Response Examples

```
200 Ok
```
```
400

{"error":"message"}
```

### GET /clone

Clone a project to the local CERNBox

Query Params

project: path of the project ("/users/d/diogo/Swan Projects/Project 1")
destination: path of where to put the copy ("Swan Projects/Project 3/")

Response Examples

```
200 Ok
```

```
406
{"error":"Name already exists"}
```

## User API

### GET /user

Searches the server's contact directory. Used in autocomplete.

Query Params

search: name to search for 

Response Examples

(the same as CERNBox result)


