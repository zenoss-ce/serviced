(function () {

    'use strict';
    angular
        .module('auth0Service', ["auth0.auth0"])
        .factory("auth0Service", ["auth0.auth0", "$timeout", "$cookies", "$cookieStore", "$location", "$http", "$notification", "miscUtils", "log",
            function auth0Service(angularAuth0, $timeout, $cookies, $cookieStore, $location, $http, $notification, utils, log) {
                var loggedIn = false;
                var userName = null;

                var setLoggedIn = function(truth, username) {
                    loggedIn = truth;
                    userName = username;
                };

                function login() {
                    debugger
                    angularAuth0.authorize();
                }
                //
                // function handleAuthentication() {
                //     angularAuth0.parseHash(function (err, authResult) {
                //         if (authResult && authResult.accessToken && authResult.idToken) {
                //             setSession(authResult);
                //             // $state.go('home');
                //         } else if (err) {
                //             $timeout(function () {
                //                 // $state.go('home');
                //             });
                //             console.log(err);
                //             alert('Error: ' + err.error + '. Check the console for further details.');
                //         }
                //     });
                // }

                // function setSession(authResult) {
                //     // Set the time that the access token will expire at
                //     let expiresAt = JSON.stringify((authResult.expiresIn * 1000) + new Date().getTime());
                //     localStorage.setItem('access_token', authResult.accessToken);
                //     localStorage.setItem('id_token', authResult.idToken);
                //     localStorage.setItem('expires_at', expiresAt);
                // }

                function logout() {
                    // Remove tokens and expiry time from localStorage
                    localStorage.removeItem('access_token');
                    localStorage.removeItem('id_token');
                    localStorage.removeItem('expires_at');
                    // $state.go('home');
                }

                //
                // function isAuthenticated() {
                //     // Check whether the current time is past the
                //     // access token's expiry time
                //     let expiresAt = JSON.parse(localStorage.getItem('expires_at'));
                //     return new Date().getTime() < expiresAt;
                // }

                /*
                 * Check whether the user appears to be logged in. Update path if not.
                 *
                 * @param {object} scope The 'loggedIn' property will be set if true
                 */
                function checkLogin($scope) {
                    $scope.dev = $cookieStore.get("ZDevMode");
                    if (loggedIn || $cookies.get("ZCPToken")) {
                        $scope.loggedIn = true;
                        $scope.user = {
                            username: $cookies.get("ZUsername")
                        };
                        return;
                    }
                    utils.unauthorized($location);
                }

                return {
                    // handleAuthentication: handleAuthentication,
                    // isAuthenticated: isAuthenticated,

                    /// methods in authService.js:
                    setLoggedIn: setLoggedIn,
                    login: login,
                    logout: logout,
                    checkLogin: checkLogin
                };
            }]);
})();
