function uploadFile(files, nr, uri) {
    // when uploading complete - redirect to new address
    if (files.length <= nr) {
        if (uri != "") {
            onUploadingComplete(uri);
        }

        return;
    }

    var currentFile = files[nr];
    if (isNotImage(currentFile.type)) {
        uploadFile(files, nr + 1, uri);
        return;
    }

    var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function () {
        if (xhr.readyState === 4) {
            var newHash = xhr.responseText;
            console.log("New hash - " + newHash);
            if (newHash.length != 64) {
                // something wrong
                console.log("Something wrong on uploading");
                alert('Oh, error on PUT file to BZZ. See log for more information.');

                return;
            }

            insertImage(files, nr, newHash, currentFile.name, function () {
                uploadFile(files, nr + 1, "/bzz:/" + newHash + "/");
            });
        }
    };
    xhr.open("PUT", uri + "imgs/" + currentFile.name, true);
    xhr.setRequestHeader('Content-Type', currentFile.type);

    readFile(currentFile, function (result) {
        xhr.send(result);
    });
}

function readFile(file, onComplete) {
    var reader = new FileReader();
    reader.onload = function (evt) {
        if (onComplete) {
            onComplete(evt.target.result)
        }
    };
    reader.readAsArrayBuffer(file);
}

function insertImage(files, nr, newHash, fileName, onComplete) {
    // insert image into index
    var img = new Image();
    img.onload = function () {
        var blur = imageToUrl(img, 5, 5);
        var thumbData = [];
        var thumbSize = 200;
        if (img.naturalWidth > img.naturalHeight) {
            // landscape thumbnail
            var h = img.naturalHeight * thumbSize / img.naturalWidth;
            thumbData[0] = imageToUrl(img, thumbSize, h);
            thumbData[1] = [thumbSize, h];
        } else if (img.naturalWidth < img.naturalHeight) {
            // portrait thumbnail
            var w = img.naturalWidth * thumbSize / img.naturalHeight;
            thumbData[0] = imageToUrl(img, w, thumbSize);
            thumbData[1] = [w, thumbSize];
        } else {
            // square
            thumbData[0] = imageToUrl(img, thumbSize, thumbSize);
            thumbData[1] = [thumbSize, thumbSize];
        }

        jQuery('#currentPreview').attr('src', thumbData[0]);
        // update index
        var imgData = [];
        imgData[0] = "imgs/" + fileName;
        imgData[1] = [img.naturalWidth, img.naturalHeight];
        imgs.data.splice(eidx, 0, {img: imgData, thumb: thumbData, blur: blur});
        if (onComplete) {
            onComplete();
        }
    };

    img.onerror = function () {
        if (onComplete) {
            onComplete();
        }
    };

    img.src = "/bzz:/" + newHash + "/imgs/" + fileName;
}

function isNotImage(type) {
    var imageType = /^image\//;
    return !imageType.test(type);
}

function onUploadingComplete(uri) {
    var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function () {
        if (xhr.readyState === 4) {
            var i = xhr.responseText;
            window.location = "/bzz:/" + i + "/" + window.location.hash;
        }
    };
    sendImages(xhr, uri);
}

function handleFiles(files) {
    showModal('Uploading photos..');
    uploadFile(files, 0, "");
}

function sendImages(xhr, uri) {
    // set up request
    xhr.open("PUT", uri + "data.json", true);
    xhr.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
    // send the collected data as JSON
    xhr.send(JSON.stringify(imgs));
}

// do it because I love jQuery
function jqueryInit() {
    // setup upload file selector
    var fileElem = jQuery("#fileElem");
    jQuery("#fileSelect").on("click", function (e) {
        if (fileElem) {
            fileElem.click();
        }

        e.preventDefault();
    });
}