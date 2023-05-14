## Server-go
API of ReviewdDB(formerly known as User and Server reviews)

# API Endpoints

## ReviewDB
## GET `/api/reviewdb/users/<discordid>/reviews`
Returns list of reviews of that user
return 51 reviews by default, if you want to get more you can add `?offset=50` to query where offset is the number of reviews you want to skip 
```json
{
	"success": true,
	"message": "",
	"hasNextPage": false,
	"reviews": [
		{
			"id": 245336,
			"sender": {
				"id": 1,
				"discordID": "287555395151593473",
				"username": "mantikafasi#4444",
				"profilePhoto": "https://cdn.discordapp.com/avatars/287555395151593473/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
				"badges": [
					{
						"name": "Admin",
						"icon": "https://cdn.discordapp.com/emojis/1040004306100826122.gif?size=128",
						"redirectURL": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
						"type": 0,
						"description": "This user is an admin of ReviewDB."
					}
				]
			},
			"star": -1,
			"comment": "Good",
			"type": 0,
			"timestamp": 1683749147
		}
	]
}
```
### Fields

> id : id of review, used for reporting and deleting

> star : unused for now (probably forever)

> comment: review of the user

> sender : User info of the user who sent the review

> > username : username

> > profilePhoto: url of users profile photo who sent the review

> > badges[ ] : List of badges user has

> > redirect_url: the url user will be redirected when clicked into badge

## PUT `/api/reviewdb/reports`

Reports the specific user
### Example body while sending request
```json
{"token" : "akd3qegd","reviewid":123}
```
returns "Response" object which contains success status and message
```json
{"success":true,"message":"Successfully reported review"}
```

## PUT `/api/reviewdb/{discordid}/reviews`
Adds review to database

Example Body Json
```json
{"token":"asdasdasd","comment":"this is pog","reviewtype":1}
```
Note: reviewtype 0 means user review and 1 means server review

### returns
```json
{"success":true,"message":"Successfully added review","updated":true}
```


## GET `/admins`
returns list of reviewdb admins
```json
[
    "1239129321312321",
    "1193128939812389"
]
```
## `/api/reviewdb/users`
Takes token as header and returns user info
"Authorization":"token"

```json
	{
	"ID": 1,
	"discordID": "123123123122313123",
	"username": "guhhhbleh#9123",
	"profilePhoto": "https://cdn.discordapp.com/avatars/123123123122313",
	"clientMod": "guhcord",
	"warningCount": 0,
	"badges": [
		{
			"name": "Admin",
			"icon": "https://cdn.discordapp.com/emojis/1040004306100826122.gif?size=128",
			"redirectURL": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			"type": 0,
			"description": "This user is an admin of ReviewDB."
		}
	],
	"banInfo": {
		"id": 1,
		"discordID": "123123123122313123",
		"reviewID": 245567,
		"reviewContent": "Explode",
		"banEndDate": "2023-05-18T10:02:33.56492Z",
		"reviewTimestamp": "2023-05-11T10:02:17.3436Z"
	},
	"lastReviewID": 244889,
	"type": 1
}
```
### Fields
> ID : id of user in database

> warningCount : number of warnings user has, if user exceeds 2 warning user will become permanently banned

> banInfo : if user is banned there will be ban info which contains ban reason and ban date

> type: type of user, 0 means user is regular, 1 means admin and -1 means user is permanently banned

> lastReviewID: last review id of user, used to notify user when someone reviews them
## `Authorization`
To authorize you have 2 options 
### First Option
 Get autorization code within discord via oauth2 modal and make a request to /URAuth endpoint with the code and client mod you are using
#### Example
```/api/reviewdb/auth?code=oauthcodeyougot&clientMod=aliucord```
### Second option
Redirect users to 
<https://discord.com/api/oauth2/authorize?client_id=915703782174752809&redirect_uri=https%3A%2F%2Fmanti.vendicated.dev%2Fapi%2Freviewdb%2Fauth&response_type=code&scope=identify>

### Response
After authorizing if authorization a "Response" object will be returned
```json
{"success":true,"token":"asdasdasd"}
```

# StupidityDB

## `/getuser?discordid=<>`
Returns the stupidity of user.
Returns integer if found, "None" if not
