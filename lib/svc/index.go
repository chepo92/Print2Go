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

<template v-if="jobRunning">

  <h3>{{ jobDescription }}</h3>
  <button v-on:click="onCancelJob" class="btn btn-danger">Cancel current job</button>
</template>
<template v-else>
  <div class="file-upload">
    <input type="file" @change="onGcodeFileSelected" />
    <br>
    <button @click="onGcodeStartUpload" class="btn btn-primary" :disabled="!this.selectedFile">Print file</button>
  </div>
</template>

</div>
<script>

new Vue({
  el: '#app',
  data: {
    jobRunning: false,
    jobDescription: "",
    selectedFile: "",
  },
  methods: {
    loadData: function () {
      jQuery.get('api/job', function (response) {
        this.jobRunning = response.job != null;
        this.jobDescription = response.status;
      }.bind(this));
    },
    onCancelJob: function() {
      var ack = prompt("Really cancel? Type 'YES' (all caps) to confirm", "");
      if (ack != "YES") {
        return;
      }
      jQuery.ajax({
        url: 'api/job',
        method: 'POST',
        data: {cancel: true},
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
            this.jobRunning = true;
            alert('print uploaded');
        },
        error: function(data) {
            alert('post failure: ' + data.responseText);
        },
      });
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
