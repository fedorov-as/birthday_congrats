$(document).ready(function () {
    $("#myForm").on("submit", function (event) {
        event.preventDefault();
        const data = {
            name: $("#name").val(),
            email: $("#email").val(),
        };
        $.ajax({
            url: "/your_server_endpoint",
            type: "post",
            data: data,
            success: function (response) {
                console.log("Данные успешно отправлены!");
            },
            error: function (error) {
                console.error("Ошибка при отправке данных: ", error);
            }
        });
    });
});