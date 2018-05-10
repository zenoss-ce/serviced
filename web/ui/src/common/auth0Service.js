/* globals Config */

(function () {

    'use strict';
    angular
        .module('auth0Service', ["auth0.auth0"])
        .factory("auth0Service", ["angularAuth0", "$timeout", "$cookies", "$cookieStore", "$location", "$http", "$notification", "miscUtils", "log",
            function auth0Service(angularAuth0, $timeout, $cookies, $cookieStore, $location, $http, $notification, utils, log) {
                var loggedIn = false;
                var userName = null;

                var setLoggedIn = function(truth, username) {
                    loggedIn = truth;
                    userName = username;
                };

                function login() {
                    angularAuth0.authorize();
                }

                function logout() {
                    // Remove tokens and expiry time from localStorage
                    localStorage.removeItem('access_token');
                    localStorage.removeItem('id_token');
                    localStorage.removeItem('expires_at');
                }

                /*
                 * Check whether the user appears to be logged in. Update path if not.
                 *
                 * @param {object} scope The 'loggedIn' property will be set if true
                 */
                function checkLogin($scope) {
                    var at = window.sessionStorage.getItem("auth0AccessToken");
                    var it  = window.sessionStorage.getItem("auth0IDToken");
                    if (at && it) {
                        $scope.loggedIn = true;
                        $scope.user = {
                            username: "successful auth0 login"
                        };
                        return;
                    }
                    utils.unauthorized($location);
                }

                return {
                    /// methods in authService.js:
                    setLoggedIn: setLoggedIn,
                    login: login,
                    logout: logout,
                    checkLogin: checkLogin
                };
            }])
        .config(config);

    config.$inject = [
        'angularAuth0Provider'
    ];

    function config(angularAuth0Provider) {
        // Initialization for the angular-auth0 library
        angularAuth0Provider.init({
            domain: window.Config.Auth0Domain,
            clientID: window.Config.Auth0ClientID,
            redirectUri: window.location.origin + "/static/auth0callback.html",
            audience: window.Config.Auth0Audience,
            responseType: "token id_token",
            scope: 'openid profile read:messages'
        });

    }
})();
