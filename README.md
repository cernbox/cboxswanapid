# cboxswanapid
SWAN API Daemon for CERNBox

# Preliminary version of the spec

## Authentication

### GET /authenticate

Protected by shibboleth.

You may need to make this request twice in case of shibboleth redirection.


Query parameters: 

```
Origin=https://swanXXX.cern.ch
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

Returns a list of all projects shared by logged in user.

Response Examples

```
200

{ "shares": [
    {"project": "SWAN_projects/SP1", 
     "path": "/eos/scratch/user/m/moscicki/SWAN_projects/SP1", 
     "shared_by": "moscicki", 
     "size": "1300"
     "inode": "10635762", 
     "shared_with": [ 
                {"permissions": "r", "created": "2017-11-07T19:45:54", "name": "moscicki", "entity": "u"}, 
                {"permissions": "r", "created": "2017-11-07T19:45:54", "name": "kubam", "entity": "u"}
              ]
    }, 
    {"project": "SWAN_projects/SP2", 
     "path": "/eos/scratch/user/m/moscicki/SWAN_projects/SP2", 
     "shared_by": "moscicki", 
     "size": "1250667"}     
     "inode": "10635763",
     "shared_with": [ 
                {"permissions": "r", "created": "2017-11-07T19:45:31", "name": "kubam", "entity": "u"}, 
                {"permissions": "r", "created": "2017-11-07T19:45:31", "name": "kuba", "entity": "u"}
              ] 
]}

```

### GET /shared

Returns all projects shared with the logged in user.

Response Examples: same as for /sharing


### GET /share

Returns details on a project shared by logged in user.

Query parameters

```
project: path of the project ("SWAN_projects/Project 1/")
```

Response Examples: same as for /sharing but contains only the chosen project entry

### PUT /share

Shares a project with specified users or groups. If project was shared with other users or group it will not longer be shared them.

Query parameters
```
project: path of the project ("SWAN_projects/Project 1/")
```

Body

```
{"share_with": [
   {"name":"moscicki", "entity":"u"}, 
   {"name":"Higgs-search-team", "entity":"egroup"} 
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

Query Parameters

```
project: path of the project ("SWAN_projects/Project 1/")
```

Response Examples

```
200 Ok
```
```
400

{"error":"message"}
```

### GET /clone

Clone a project to the local CERNBox of the authenticated user.

Query Params

```
project: path of the project ("SWAN_projects/Project 1")
sharer: name of the user who shared the project
destination: new name of the project ("SWAN_projects/Project 3/")
```

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


