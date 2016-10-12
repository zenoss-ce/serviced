/* hostIconDirective
 * directive for displaying status of a host
 */
(function() {
    'use strict';

    angular.module('hostIcon', [])
    .directive('hostIcon', [function() {
        var template = `
              <div class="host-stat-icon" ng-class="vm.getHostStatusClass()">
                  <div style="white-space: nowrap;"><i class="glyphicon" ng-class="vm.getHostActiveStatusClass()"></i> Active</div>
                  <div style="white-space: nowrap;"><i class="glyphicon" ng-class="vm.getHostAuthStatusClass()"></i> Auth</div>
              </div>`;
      
        class Controller {
            constructor(){
                // DO THINGS
            }
            getHostStatusClass(){
                let {active, authed} = this._getHostStatus();

                // stuff hasnt loaded, so unknown
                if(active === null && authed === null){
                    return "unknown";
                }

                // connected and authenticated
                if(active && authed){
                    return "passed";

                // connected but not yet authenticated
                } else if(active && !authed){
                    // TODO - something more clearly related to auth
                    return "unknown";

                // not connected
                } else {
                    return "failed";
                }
            }

            _getHostStatus(){
                if(!this.host){
                    return {active: null, authed: null};
                }

                let status = this.getHostStatus(this.host.id);

                if(!status){
                    return {active: null, authed: null};
                }

                let active = status.Active,
                    authed = status.Authenticated;

                return {active, authed};
            }

            getHostActiveStatusClass(){
                let {active, authed} = this._getHostStatus(),
                    status;

                if(active === true){
                    status = "glyphicon-ok";
                } else if(active === false){
                    status = "glyphicon-exclamation-sign";
                } else {
                    status = "glyphicon-question-sign";
                }

                return status;
            }

            getHostAuthStatusClass(){
                let {active, authed} = this._getHostStatus(),
                    status;

                if(authed === true){
                    status = "glyphicon-ok";
                } else if(authed === false){
                    status = "glyphicon-exclamation-sign";
                } else {
                    status = "glyphicon-question-sign";
                }

                return status;
            }


        }
      
        return {
            restrict: "EA",
            scope: {
                host: "=",
                getHostStatus: "="
            },
            controller: Controller,
            controllerAs: "vm",
            bindToController: true,
            template: template,
            link: function(scope, element, attrs) {
                element.addClass("host-stat-icon");
            }

        };
  }]);

}());
