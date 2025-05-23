Этот репозиторий содержит мои реализации некоторых контейнеров и утилит из стандартной библиотеки C++ (STL), выполненные в рамках курса по C++ на ФПМИ МФТИ.

### Реализованные классы
1. big_integer.h: **`BigInteger`** - класс для работы с большими целыми числами произвольной длины
   - Поддерживает базовые арифметические операции (+, -, *, /, %)
   - Реализованы операции сравнения
   - Поддержка ввода/вывода
   - Литералы

2. big_integer.h: **`Rational`** - рациональные числа
   - Арифметические операции с дробями
   - Приведение к десятичной дроби с заданной точностью
   - Операции сравнения
  
3. geometry.h: **`Geometry`** - геометрические примитивы и фигуры
   - Точки, векторы, линии
   - Фигуры: эллипс, круг, многоугольник, прямоугольник, квадрат, треугольник
   - Операции: поворот, отражение, масштабирование
   - Проверка свойств (подобие, конгруэнтность)
   - Наследование и вирутальные функции
  
4. string.h: **`String`** - строка с динамическим буфером
   - Базовые строковые операции
   - Поиск подстрок
   - Ввод/вывод

5. deque.h: **`Deque`** - двусторонняя очередь
   - Поддержка итераторов
   - Вставка/удаление с обоих концов
   - Доступ по индексу за O(1)
   - Обработка исключений

6. stackallocator.h: **`List` со `StackAllocator`** - двусвязный список
   - Аллокатор на стеке
   - Поддержка итераторов
   - Вставка/удаление элементов
   - Обработка исключений

7. smart_pointers.h: **`SharedPtr` и `WeakPtr`** - умные указатели
   - Подсчет ссылок
   - Поддержка enable_shared_from_this
   - Кастомные аллокаторы и делитеры
   - move семантика
   - Обработка исключений

8. unordered_map.h: **`UnorderedMap`** - хэш-таблица
   - Разрешение коллизий методом цепочек
   - Поддержка рехешинга
   - Поддержка итераторов
   - move семантика
   - Обработка исключений
