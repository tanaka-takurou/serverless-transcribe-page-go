$(document).ready(function() {
  var
    $headers     = $('body > h1'),
    $header      = $headers.first(),
    ignoreScroll = false,
    timer;

  $(window)
    .on('resize', function() {
      clearTimeout(timer);
      $headers.visibility('disable callbacks');

      $(document).scrollTop( $header.offset().top );

      timer = setTimeout(function() {
        $headers.visibility('enable callbacks');
      }, 500);
    });
  $headers
    .visibility({
      once: false,
      checkOnRefresh: true,
      onTopPassed: function() {
        $header = $(this);
      },
      onTopPassedReverse: function() {
        $header = $(this);
      }
    });
});

var ChangeAudio = function() {
  const file = $('#mp3').prop('files')[0];
  toBase64(file).then(onConverted());
}

var toBase64 = function(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result);
    reader.onerror = error => reject(error);
  });
}

var onConverted = function() {
  return function(v) {
    App.mp3data = v;
    $('#preview').attr('src', v);
  }
}

var SubmitForm = function() {
  $("#submit").addClass('disabled');
  var mp3 = App.mp3data;
  var action  = $("#action").val();
  if (!mp3) {
    $("#submit").removeClass('disabled');
    $("#warning").text("MP3 is Empty").removeClass("hidden").addClass("visible");
    return false;
  }
  const data = {action, mp3};
  request(data, (res)=>{
    $("#info").removeClass("hidden").addClass("visible");
    App.jobName = res.message
    CheckProgress();
  }, (e)=>{
    console.log(e.responseJSON.message);
    $("#warning").text(e.responseJSON.message).removeClass("hidden").addClass("visible");
    $("#submit").removeClass('disabled');
  });
};

var CheckProgress = function() {
  var action  = "checkprogress";
  var name = App.jobName;
  if (!name) {
    $("#warning").text("Job Name is Empty").removeClass("hidden").addClass("visible");
    return false;
  }
  const data = {action, name};
  request(data, (res)=>{
    if (res.message == "COMPLETED") {
      GetTranscription();
    } else if (res.message == "FAILED") {
      $("#warning").text("Error: TranscriptionJob Failed").removeClass("hidden").addClass("visible");
    } else {
      setTimeout(function() {
        CheckProgress();
      }, 10000);
    }
  }, (e)=>{
    console.log(e.responseJSON.message);
    $("#warning").text(e.responseJSON.message).removeClass("hidden").addClass("visible");
    $("#submit").removeClass('disabled');
  });
};

var GetTranscription = function() {
  var action  = "gettranscription";
  var name = App.jobName;
  if (!name) {
    $("#warning").text("Job Name is Empty").removeClass("hidden").addClass("visible");
    return false;
  }
  const data = {action, name};
  request(data, (res)=>{
    $("#result_json").text(res.message);
    $("#result").removeClass("hidden").addClass("visible");
  }, (e)=>{
    console.log(e.responseJSON.message);
    $("#warning").text(e.responseJSON.message).removeClass("hidden").addClass("visible");
    $("#submit").removeClass('disabled');
  });
};

var request = function(data, callback, onerror) {
  $.ajax({
    type:          'POST',
    dataType:      'json',
    contentType:   'application/json',
    scriptCharset: 'utf-8',
    data:          JSON.stringify(data),
    url:           {{ .Api }}
  })
  .done(function(res) {
    callback(res);
  })
  .fail(function(e) {
    onerror(e);
  });
};

var App = { mp3data: null, jobName: null };
