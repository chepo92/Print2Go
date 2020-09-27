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
        <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" crossorigin="anonymous">

        <script src="https://cdn.jsdelivr.net/npm/vue@2.6.11"></script>
        <script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/2.2.4/jquery.min.js"></script>
        <title>gfeeder status</title>
    </head>
    <body>
<div class="container" id="app">
<br>

<template v-if="jobRunning">
  <h3>{{ jobDescription }}</h3>
  <button v-on:click="onCancelJob">Cancel current job</button>
</template>
<template v-else>
  <div class="file-upload">
    <input type="file" @change="onGcodeFileSelected" />
    <br>
    <button @click="onGcodeStartUpload" class="upload-button" :disabled="!this.selectedFile">Print file</button>
  </div>
</template>

</div>
<script>

function normalize(x) {
  if (x < 1024) {
   return x + ' Bytes';
  }
  if (x < 4096) {
   return (x/1024).toFixed(2) + ' KB';
  }
  return (x/1024/1024).toFixed(2) + ' MB';
}

function duration(x) {
  d = ((new Date()) - x)/1000;
  if (d < 60) {
    return d.toFixed(0) + ' sec';
  }
  return (d/60).toFixed(2) + ' min';
}

new Vue({
  el: '#app',
  data: {
    jobRunning: false,
    jobDescription: "",
    selectedFile: "",
  },
  methods: {
    loadData: function () {
      $.get('api/job', function (response) {
        this.jobRunning = response.job != null;
        this.jobDescription = response.status;
      }.bind(this));
    },
    onCancelJob: function() {
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
