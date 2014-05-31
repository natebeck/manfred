if (window.gamePageConfig && window.gamePageConfig.id) {
    $('table').on('click', 'a.refresh', function (e) {
        var $this = $(this),
            twitchId = $this.attr('data-twitch-id');

        e.preventDefault();

        $.ajax({
            url: '/game/' + gamePageConfig.id + '/player',
            method: 'DELETE',
            contentType: 'application/json; charset=utf-8',
            data: JSON.stringify({ userKey: twitchId }),
            success: function () {
                window.location.reload();
            }
        });
    });
}
