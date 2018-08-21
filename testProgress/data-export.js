server_addr = window.location.hostname;
if (!server_addr) {
    server_addr = 'localhost';
}

CheckStorage = function () {
    const url = 'http://' + server_addr + ':2223/export/storage/status';
    const xhr = new XMLHttpRequest();
    xhr.open("GET", url, false);
    xhr.send();
    let isConnected = false;
    if (xhr.status === 200) {
        const response = JSON.parse(xhr.response);
        isConnected = response.storage_installed;
    } else {
        console.log("status = " + xhr.status);
        isConnected = false;
    }
    return isConnected;
};

StartExport = function (fileList) {
    let request = {
        target_files: []
    };
    angular.forEach(fileList, function(file) {
        request.target_files.push(file.fullpath);
    });
     //StartExport(JSON.stringify(request));
    const url = 'http://' + server_addr + ':2223/export/storage/execute';
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url, true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function (e) {
        if (xhr.readyState === 4) {
            if (xhr.status === 200) {
                console.log(xhr.responseText);
            } else {
                console.error(xhr.responseText);
            }
        }
    };
    xhr.onerror = function (e) {
        console.error(xhr.responseText);
    };
    xhr.send(JSON.stringify(request));
};

angular.module('olive.waveData.export', ['ngAnimate', 'ngSanitize', 'ui.bootstrap']);
angular.module('olive.waveData.export').controller('ExportModalCtrl', function ($scope, $uibModal, $log) {
    let em = this;
    em.data = {
        exportingProgress: 0,
        exportingMessage: "Processing",
        exportingFile: "",
        wsConnection: null,
    };
    em.open = function (fileList) {
        let modalInstance = $uibModal.open({
            animation: true,
            ariaLabelledBy: 'modal-title',
            ariaDescribedBy: 'modal-body',
            templateUrl: 'dataExportProgress.html',
            controller: 'ModalInstanceCtrl',
            controllerAs: 'em',
            backdrop: 'static',
            scope: $scope,
            resolve: {
                data: function () {
                    return em.data;
                }
            }
        });
        modalInstance.result.then(function () {
            em.data.wsConnection.close();
        }, function () {
            em.data.wsConnection.close();
        });
        const url = 'ws://' + server_addr + ':2222/export/storage/event';
        em.data.wsConnection = new WebSocket(url);

        em.data.wsConnection.onmessage = function (event) {
            const msg = JSON.parse(event.data);
            console.log(msg);
            em.data.exportingProgress = msg.total_progress * 100;
            em.data.exportingFile = msg.current_export;
            if (msg.total_progress >= 1) {
                em.data.exportingMessage = "Done";
            }
            $scope.$apply();
        };
        if (CheckStorage()) {
            StartExport(fileList)
        } else {
            em.data.exportingMessage = "Storage not connected";
            em.data.exportingProgress = 0;
        }
    };

});

angular.module('olive.waveData.export').controller('ModalInstanceCtrl', function ($uibModalInstance, data) {
    let em = this;
    em.data = data;
    em.ok = function () {
        em.data.exportingProgress = 0;
        em.data.exportingMessage = "Processing";
        em.data.exportingFile = "";
        $uibModalInstance.close();
    };
    em.cancel = function () {
        em.data.exportingProgress = 0;
        em.data.exportingMessage = "Processing";
        em.data.exportingFile = "";
        $uibModalInstance.dismiss('cancel');
        $uibModalInstance.close();
    };
});
