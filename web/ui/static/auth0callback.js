
var webAuth = new auth0.WebAuth({
    domain: 'zenoss-dev.auth0.com',
    clientID: 'xQF6jCIx6ZynvlvzT8ZWWrbOswcgCwH9',
    redirectUri: window.location + "/auth0callback.html",
    audience: 'https://dev.zing.ninja',
    responseType: "token id_token",
    scope: 'openid profile read:messages'
});


webAuth.parseHash(function (err, result) {
   console.log("result: " + JSON.stringify(result));
   console.log("error: " + JSON.stringify(err));
   if (err) {
       console.error("Unable to authenticate: " + err);
       webAuth.authorize();
   } else if (result && result.idToken && result.accessToken) {
       window.sessionStorage.setItem("auth0AccessToken", result.accessToken);
       window.sessionStorage.setItem("auth0IDToken", result.idToken);
       console.log('window.location.origin = ' + window.location.origin);
       window.location = window.location.origin;
   }
});

