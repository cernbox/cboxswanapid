# cboxswanapid
SWAN API Daemon for CERNBox

## Authentication

### GET /authenticate

Protected by shibboleth.


Query parameters: 

```
Origin=https://swanXXX.example.org
```

This parameter must be of type _swan*.example.org_

Response headers

```
Content-Security-Policy: frame-ancestors https://swanXXX.example.org
```

Accessed through an iFrame. Hence it sets the header Content-Security-Policy to be possible to open it as an iFrame.

Returns a page with a script that calls parent.postMessage(...) (https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage). This call should send a token with expire date (ISO format).

Response Examples

```
<script>parent.postMessage({"authtoken":"xxxx","expire":"2017-06-20 13:00:00"}, 'https://swanXXX.example.org');</script>
```

## Security

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
 
 Access-Control-Allow-Origin: https://swanXXX.example.org
 Access-Control-Allow-Methods: GET, POST, PUT, DELETE (depending on the endpoint)
 Access-Control-Allow-Headers: Authorization
 
 ```
 
In the Allow-Methods, the list should contain all the methods allowed on that endpoint, so that the browser can cache 
this reply.


## Sharing API
 

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
                {"permissions": "r", "created": "2017-11-07T19:45:54", "name": "moscicki", "entity": "u", "display_name": "Jakub Moscicki"}, 
                {"permissions": "r", "created": "2017-11-07T19:45:54", "name": "kubam", "entity": "u", "display_name": "Jakub Moscicki"}
              ]
    }, 
    {"project": "SWAN_projects/SP2", 
     "path": "/eos/scratch/user/m/moscicki/SWAN_projects/SP2", 
     "shared_by": "moscicki", 
     "size": "1250667"}     
     "inode": "10635763",
     "shared_with": [ 
                {"permissions": "r", "created": "2017-11-07T19:45:31", "name": "kubam", "entity": "u", "display_name": "Jakub Moscicki"}, 
                {"permissions": "r", "created": "2017-11-07T19:45:31", "name": "kuba", "entity": "u", "display_name": "Jakub Moscicki"}
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

Shares a project with specified users or groups. If project was shared with other users or group not present in this list, it will not longer be shared with them.

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
Entity can be "egroup", for egroups, "g", for unixgroup, and "u" for all other user accounts (primary, secondary and service).

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

### POST /clone

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

## Directory API

### GET /search?filter=`<filter>`

Searches the server's contact directory. Used in autocomplete.
The account_type key can have the following values: primary, secondary, service, egroup and unixgroup.

Query Params

```
filter: name to search for. If name is prefixed by *a*: the result will also include service and secondary accounts.
If the prefix g: is used, only unix groups will be shown.
```

Response Examples


```
200

[
    {
        "account_type": "primary",
        "cn": "casallab",
        "display_name": "Jorge Casal Labrador",
        "dn": "CN=casallab,OU=Users,OU=Organic Units,DC=cern,DC=ch",
        "mail": "jorge.casal.labrador@example.org"
    },
    {
        "account_type": "primary",
        "cn": "gonzalhu",
        "display_name": "Hugo Gonzalez Labrador",
        "dn": "CN=gonzalhu,OU=Users,OU=Organic Units,DC=cern,DC=ch",
        "mail": "hugo.gonzalez.labrador@example.org"
    },
    {
        "account_type": "egroup",
        "cn": "cernbox-project-labradorprojecttest-writers",
        "display_name": "cernbox-project-labradorprojecttest-writers (CERNBOX PROJECT LABRADORPROJECTTEST WRITERS)",
        "dn": "CN=cernbox-project-labradorprojecttest-writers,OU=e-groups,OU=Workgroups,DC=cern,DC=ch",
        "mail": "cernbox-project-labradorprojecttest-writers@example.org"
    },
    {
        "account_type": "egroup",
        "cn": "cernbox-project-labradorprojecttest-readers",
        "display_name": "cernbox-project-labradorprojecttest-readers (CERNBOX PROJECT LABRADORPROJECTTEST READERS)",
        "dn": "CN=cernbox-project-labradorprojecttest-readers,OU=e-groups,OU=Workgroups,DC=cern,DC=ch",
        "mail": "cernbox-project-labradorprojecttest-readers@example.org"
    },
    {
        "account_type": "egroup",
        "cn": "cernbox-project-labradorprojecttest-admins",
        "display_name": "cernbox-project-labradorprojecttest-admins (CERNBOX PROJECT LABRADORPROJECTTEST ADMINS)",
        "dn": "CN=cernbox-project-labradorprojecttest-admins,OU=e-groups,OU=Workgroups,DC=cern,DC=ch",
        "mail": "cernbox-project-labradorprojecttest-admins@example.org"
    }
]

```


