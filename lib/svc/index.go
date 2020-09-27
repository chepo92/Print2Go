package svc

import (
	"net/http"
)

func (svc *Svc) indexPage(w http.ResponseWriter) {
	x := `<!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <meta http-equiv="X-UA-Compatible" content="ie=edge">

<script src="https://cdn.jsdelivr.net/npm/vue@2.6.12"></script>
<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.5.2/css/bootstrap.min.css" integrity="sha384-JcKb8q3iqJ61gNV9KGb8thSsNjpSL0n8PARn9HuZOnIxN0hoP+VmmDGMN5t9UJ0Z" crossorigin="anonymous">
<script src="https://code.jquery.com/jquery-3.5.1.min.js" integrity="sha384-ZvpUoO/+PpLXR1lu4jmpXWu80pZlYUAfxl5NsBMWOEPSjUn/6Z/hRTt8+pR6L4N2" crossorigin="anonymous"></script>
<script src="https://cdn.jsdelivr.net/npm/popper.js@1.16.1/dist/umd/popper.min.js" integrity="sha384-9/reFTGAW83EW2RDu2S0VKaIzap3H66lZH81PoYlFhbGU+6BZp6G7niu735Sk7lN" crossorigin="anonymous"></script>
<script src="https://stackpath.bootstrapcdn.com/bootstrap/4.5.2/js/bootstrap.min.js" integrity="sha384-B4gt1jrGC7Jh4AgTPSdUtOBvfO8shuf57BaghqFfPlYxofvL8/KUEfYiJOMMV+rV" crossorigin="anonymous"></script>

        <title>gfeeder status</title>
    </head>
    <body>
<div class="container" id="app">
<br>

<div class="modal fade" id="cancelDialog" tabindex="-1" role="dialog" aria-hidden="true">
  <div class="modal-dialog" role="document">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title" id="exampleModalLabel">Cancel print operation</h5>
        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <div class="modal-body">
Do you really want to cancel the ongoing print?
      </div>
      <div class="modal-footer">
        <button type="button" class="btn btn-secondary" data-dismiss="modal">Close</button>
        <button type="button" class="btn btn-danger" v-on:click="onCancelJob" data-dismiss="modal">Yes, cancel</button>
      </div>
    </div>
  </div>
</div>

<div class="modal fade" id="uploadDialog" tabindex="-1" role="dialog" aria-hidden="true">
  <div class="modal-dialog" role="document">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title">Upload gcode</h5>
        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <div class="modal-body">
Select gcode to upload:
    <input type="file" @change="onGcodeFileSelected" />
      </div>
      <div class="modal-footer">
        <button type="button" class="btn btn-secondary" data-dismiss="modal">Cancel</button>
    <button v-on:click="onGcodeStartUpload" class="btn btn-primary" :disabled="!this.selectedFile">Upload and print</button>
      </div>
    </div>
  </div>
</div>


<template v-if="jobRunning">
<div class="progress">
  <div class="progress-bar progress-bar-striped progress-bar-animated" role="progressbar" style="width: 100%" aria-valuenow="100" aria-valuemin="0" aria-valuemax="100"></div>
</div>
Printer is currently working: {{ jobDescription }}
<br><br>
<button type="button" class="btn btn-danger" data-toggle="modal" data-target="#cancelDialog">Cancel print</button>

<hr>
<div v-for="item in jobLogBuff">
<div>{{ item }}</div>
</di>


</template>
<template v-else>
<div class="alert alert-primary" role="alert">Printer is idle.</div>

<ul class="list-group">
<li class="list-group-item"><button type="button" class="btn btn-primary" data-toggle="modal" data-target="#uploadDialog">Upload & Print</button></li>
<li class="list-group-item"><button v-on:click="onBuiltinGcode('z-axis')" type="button" class="btn btn-primary">Lift Z-Axis</button></li>
<li class="list-group-item"><button v-on:click="onBuiltinGcode('heat')" type="button" class="btn btn-primary">Heat extruder</button></li>
<li class="list-group-item"><button v-on:click="onBuiltinGcode('f-move')" type="button" class="btn btn-primary">Move filament</button></li>
<li class="list-group-item"><button v-on:click="onBuiltinGcode('reset')" type="button" class="btn btn-primary">Reset printer</button></li>
</ul>
</template>

</div>
<script>

new Vue({
  el: '#app',
  data: {
    jobRunning: false,
    jobDescription: "",
    jobLogBuff: [],
    selectedFile: "",
  },
  methods: {
    loadData: function () {
      jQuery.get('api/job', function (response) {
        this.jobRunning = response.job != null;
        this.jobDescription = response.status;
        this.jobLogBuff = response.buffer;
      }.bind(this));
    },
    onCancelJob: function() {
      jQuery.ajax({
        url: 'api/job',
        method: 'POST',
        data: {cancel: true},
      });
    },
    onBuiltinGcode: function(arg) {
      jQuery.ajax({
        url: 'api/gcode/action',
        data: { action: arg },
      });
    },
    onGcodeFileSelected: function(e) {
      this.selectedFile = e.target.files[0];
    },
    onGcodeStartUpload: function() {
      var data = new FormData();
      data.append("print", true);
      data.append("file", this.selectedFile);
      this.selectedFile = "";
      jQuery.ajax({
        url: 'api/files/local',
        enctype: 'multipart/form-data',
        contentType: false,
        data: data,
        processData: false,
        method: 'POST',
        success: function(data){
            this.selectedFile = "";
            $('#uploadDialog').modal('hide');
        },
        error: function(data) {
            alert('post failure: ' + data.responseText);
        },
      });
      return true;
    },
  },
  mounted: function () {
    this.loadData();
    setInterval(function () {
      this.loadData();
    }.bind(this), 800);
  }
});
</script>
</body>
</html>
`
	w.Write([]byte(x))
}
