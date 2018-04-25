(function () {

    'use strict';
    angular
        .module('auth0Service', ["auth0.auth0"])
        .factory("auth0Service", ["angularAuth0", "$timeout", "$cookies", "$cookieStore", "$location", "$http", "$notification", "miscUtils", "log",
            function auth0Service(angularAuth0, $timeout, $cookies, $cookieStore, $location, $http, $notification, utils, log) {
                var loggedIn = false;
                var userName = null;

                // const CLIENT_DOMAIN = "zenoss-dev.auth0.com";
                // const CLIENT_ID = "xQF6jCIx6ZynvlvzT8ZWWrbOswcgCwH9";
                //
                // angularAuth0.init({
                //     domain: CLIENT_DOMAIN,
                //     clientID: CLIENT_ID
                // });

                var setLoggedIn = function(truth, username) {
                    loggedIn = truth;
                    userName = username;
                };

                function login() {
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
            }])
        .config(config);

    config.$inject = [
        // '$stateProvider',
        // '$locationProvider',
        // '$urlRouterProvider',
        'angularAuth0Provider'
    ];

    // function config($stateProvider, $locationProvider, $urlRouterProvider,angularAuth0Provider) {
    //
    //     $stateProvider
    //         .state('home', {
    //             url: '/',
    //             controller: 'HomeController',
    //             templateUrl: 'app/home/home.html',
    //             controllerAs: 'vm'
    //         })
    //         .state('callback', {
    //             url: '/callback',
    //             controller: 'CallbackController',
    //             templateUrl: 'app/callback/callback.html',
    //             controllerAs: 'vm'
    //         });
    //
    //     // Initialization for the angular-auth0 library
    //     angularAuth0Provider.init({
    //         clientID: 'xQF6jCIx6ZynvlvzT8ZWWrbOswcgCwH9',
    //         domain: 'zenoss-dev.auth0.com',
    //         responseType: 'token id_token',
    //         audience: 'https://zenoss-dev.auth0.com/userinfo',
    //         redirectUri: 'http://localhost:3000/callback',
    //         scope: 'openid'
    //     });
    //
    //     $urlRouterProvider.otherwise('/');
    //
    //     $locationProvider.hashPrefix('');
    //
    //     /// Comment out the line below to run the app
    //     // without HTML5 mode (will use hashes in routes)
    //     $locationProvider.html5Mode(true);
    // }
    function config(angularAuth0Provider) {

        // Initialization for the angular-auth0 library
        angularAuth0Provider.init({
            clientID: 'xQF6jCIx6ZynvlvzT8ZWWrbOswcgCwH9',
            domain: 'zenoss-dev.auth0.com',
            responseType: 'token id_token',
            audience: 'https://zenoss-dev.auth0.com/userinfo',
            // redirectUri: 'http://localhost:3000/callback',
            redirectUri: 'http://localhost/apps',
            scope: 'openid'
        });

    }
})();
