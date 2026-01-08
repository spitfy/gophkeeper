# Регистрация
gophkeeper register myuser "strongpassword" --terms

# Вход
gophkeeper login myuser "strongpassword"

# Список записей
gophkeeper records list

# Подробная информация о записи
gophkeeper records get 123 --decrypt

# Создание записи
gophkeeper records create --type login --data "encrypted" --meta '{"title":"Email"}'

# Обновление записи
gophkeeper records update 123 --type login --data "newencrypted" --meta '{"title":"Updated"}'

# Удаление записи
gophkeeper records delete 123