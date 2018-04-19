/* ChallengeController
 * challenge (login) page
 */

(function() {
    'use strict';
    controlplane.controller("ChallengeController",["$scope", "$location", "$notification", "$translate", "auth0Service", "resourcesFactory",
    function($scope, $location, $notification, $translate, auth0Service, resourcesFactory) {
        // debugger;
        if(navigator.userAgent.indexOf("Trident") > -1 && navigator.userAgent.indexOf("MSIE 7.0") > -1){
            $notification.create("", $translate.instant("compatibility_mode"), $("#loginNotifications")).warning(false);
        }
        // const AUTH0_DOMAIN =  "zenoss-dev.auth0.com";
        // const AUTH0_CLIENT_ID = "xQF6jCIx6ZynvlvzT8ZWWrbOswcgCwH9";

        auth0Service.login();

        $scope.$emit("ready");

        $scope.version = "";
        resourcesFactory.getVersion().success(function(data){
            $scope.version = data.Version;
        });


    }]);
})();
