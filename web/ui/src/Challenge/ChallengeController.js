/* ChallengeController
 * challenge (login) page
 */

(function() {
    'use strict';
    controlplane.controller("ChallengeController",["$scope", "$location", "$notification", "$translate", "authService", "resourcesFactory",
    function($scope, $location, $notification, $translate, authService, resourcesFactory) {
        if(navigator.userAgent.indexOf("Trident") > -1 && navigator.userAgent.indexOf("MSIE 7.0") > -1){
            $notification.create("", $translate.instant("compatibility_mode"), $("#loginNotifications")).warning(false);
        }

        authService.auth0login();

        $scope.$emit("ready");

        $scope.version = "";
        resourcesFactory.getVersion().success(function(data){
            $scope.version = data.Version;
        });
    }]);
})();
