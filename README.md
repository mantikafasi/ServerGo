## Server-go
API of ReviewdDB(formerly known as User and Server reviews)

# API Endpoints

## ReviewDB
## `/getUserReviews?discordid=<discordid>`
Returns list of reviews of that user
```json
[{"id":225528,"star":-1,"senderuserid":3775,"comment":"This is a test review","reviewtype":0,"isSystemMessage":false,"senderdiscordid":"287555395151593473","username":"mantikafasi#4444","profile_photo":"https://cdn.discordapp.com/avatars/287555395151593473/c4b7353e759983f5a3d686c7937cfab7.png?size=128","badges":[{"badge_name":"Admin","badge_icon":"https://cdn.discordapp.com/emojis/1040004306100826122.gif?size=128","redirect_url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","badge_type":0,"badge_description":"This user is an admin of ReviewDB."}]}]
```
### Fields

> id : id of review, used for reporting and deleting

> star : unused for now (probably forever)

> senderuserid : User id of person who sent the review 

> comment: review of the user

> username : username

> profile_photo: url of users profile photo who sent the review

> badges[ ] : List of badges user has

> > redirect_url: the url user will be redirected when clicked into badge

## `/reportReview`
Reports the specific user
### Example body while sending request
```json
{"token" : "akd3qegd","reviewid":123}
```
returns "Successfully Reported Review" if successful, error string if not

## `/addUserReview`
Adds review to database

Example Body Json
```json
{"userid":1293812321,"token":"asdasdasd","comment":"this is pog","reviewtype":1}
```
### Fields
> userid : discordid of user thats been reviewed
> 
> token : token of user that is reviewing
> 
> comment: yes
> 
> reviewtype: reviewtype of review, 0 means its a user review and 1 means its server review

### returns
"Added your review" if review is successfully added , "Updated your review" if review is updated , or error string if there is a error 

## `/reportReview`
Reports the specific user
### Example body while sending request
```json 
{"token" : "akd3qegd","reviewid":123} 
```
returns "Successfully Reported Review" if successful, error string if not

## `Authroization`
To authorize you have 2 options 
### First Option
 Get autorization code within discord via oauth2 modal and make a request to /URAuth endpoint with the code and client mod you are using
#### Example
```/URAuth?code=oauthcodeyougot&clientMod=aliucord```
### Second option
Redirect users to 
<https://discord.com/api/oauth2/authorize?client_id=915703782174752809&redirect_uri=https%3A%2F%2Fmanti.vendicated.dev%2FURauth&response_type=code&scope=identify>

### Response
After authorizing if authorization is successful it will redirect to 

/receiveToken/\<token\> 

and show token to user

if some error happens in authorization user will be redirected to /error

# StupidityDB

## `/getuser?discordid=<>`
Returns the stupidity of user.
Returns integer if found, "None" if not
