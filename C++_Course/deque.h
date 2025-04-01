#include <vector>
#include <stdexcept>

template <typename T>
class Deque {
private:
  const size_t inner_size = 16;
  std::vector<T*> array;
  size_t filled_size;
  std::pair<size_t, size_t> start;
  std::pair<size_t, size_t> finish;

  void swap(Deque& other);
  void increase(size_t out_size);

public:
  template <bool is_const>
  struct general_iterator;
  using iterator = general_iterator<false>;
  using const_iterator = general_iterator<true>;
  using reverse_iterator = std::reverse_iterator<iterator>;
  using const_reverse_iterator = std::reverse_iterator<const_iterator>;

  explicit Deque(): array(1), filled_size(0), start(0, 0), finish(0, 0) {
    array[0] = reinterpret_cast<T*>(new char[inner_size * sizeof(T)]);
  }

  Deque(const Deque& other);
  Deque(size_t number);
  Deque(size_t number, const T& element);
  ~Deque();

  size_t size() const {
    return filled_size;
  }

  Deque& operator=(const Deque& other);
  T& operator[](size_t index);
  const T& operator[](size_t index) const;
  T& at(size_t index);
  const T& at(size_t index) const;
  void push_back(const T& element);
  void push_front(const T& element);
  void pop_back();
  void pop_front();

  iterator begin() {
    return iterator(start, &array[start.first]);
  }
  const_iterator begin() const {
    return const_iterator(start, &array[start.first]);
  }
  const_iterator cbegin() const {
    return const_iterator(start, &array[start.first]);
  }
  iterator end() {
    return iterator(finish, &array[finish.first]);
  }
  const_iterator end() const {
    return const_iterator(finish, &array[finish.first]);
  }
  const_iterator cend() const {
    return const_iterator(finish, &array[finish.first]);
  }
  reverse_iterator rbegin() {
    return reverse_iterator(end());
  }
  const_reverse_iterator crbegin() const {
    return const_reverse_iterator(cend());
  }
  const_reverse_iterator rbegin() const {
    return cbegin();
  }
  reverse_iterator rend() {
    return reverse_iterator(begin());
  }
  const_reverse_iterator crend() const {
    return const_reverse_iterator(cbegin());
  }
  const_reverse_iterator rend() const {
    return crend();
  }

  void insert(iterator iter, T element);
  void erase(iterator iter);
};

std::pair<size_t, size_t> change_position(std::pair<int, int> position, int delta) {
  if (delta >= 0) {
    position.first += (position.second + delta) / 16;
    position.second = (position.second + delta) % 16;
  } else if (position.second + delta < 0) {
    position.first += (position.second + delta) / 16 - 1;
    position.second = 16 + (position.second + delta) % 16;
  } else {
    position.second = position.second + delta;
  }
  return position;
}

template <typename T>
void Deque<T>::swap(Deque<T>& other) {
  std::swap(array, other.array);
  std::swap(filled_size, other.filled_size);
  std::swap(start, other.start);
  std::swap(finish, other.finish);
}

template <typename T>
void Deque<T>::increase(size_t out_size) {
  Deque new_deque;
  for (size_t i = 0; i < out_size; ++i) {
    try {
      new_deque.array.push_back(reinterpret_cast<T*>(new char[inner_size * sizeof(T)]));
    } catch (...) {
      for (size_t j = 0; j < i; ++j) {
        delete[] reinterpret_cast<char*>(new_deque.array[j]);
      }
      throw;
    }
  }
  size_t out_start = start.first;
  size_t out_finish = change_position(finish, -1).first + 1;
  for (size_t i = (out_size - (out_finish - out_start + 1) + 1) / 2;
       out_start < out_finish; ++i) {
    std::swap(new_deque.array[i], array[out_start]);
    ++out_start;
  }
  new_deque.start.first = (out_size - (out_finish - start.first + 1) + 1) / 2;
  new_deque.start.second = start.second;
  new_deque.finish.first = new_deque.start.first + (finish.first - start.first);
  new_deque.finish.second = finish.second;
  new_deque.filled_size = filled_size;
  filled_size = 0;

  swap(new_deque);
}

template <typename T>
Deque<T>::Deque(const Deque<T>& other): array(2 * (other.size() / inner_size + 1)),
                                        filled_size(other.size()),
                                        start(array.size() / 4 + 1, 0) {
  for (size_t i = 0; i < array.size(); ++i) {
    try {
      array[i] = reinterpret_cast<T*>(new char[inner_size * sizeof(T)]);
    } catch (...) {
      for (size_t j = 0; j < i; ++j) {
        delete[] reinterpret_cast<char*>(array[j]);
      }
      throw;
    }
  }

  std::pair<size_t, size_t> current(start);
  for (size_t i = 0; i < other.size(); ++i) {
    try {
      new(array[current.first] + current.second) T(other[i]);
    } catch (...) {
      for (size_t j = 0; j < i; ++j) {
        current = change_position(current, -1);
        (array[current.first] + current.second)->~T();
      }
      for(size_t j = 0; j < array.size(); ++j) {
        delete[] reinterpret_cast<char*>(array[j]);
      }
      throw;
    }
    current = change_position(current, 1);
  }
  finish = current;
}

template <typename T>
Deque<T>::Deque(size_t number): array(2 * (number / inner_size + 1)),
                                filled_size(number),
                                start(array.size() / 4 + 1, 0) {
  for (size_t i = 0; i < array.size(); ++i) {
    try {
      array[i] = reinterpret_cast<T*>(new char[inner_size * sizeof(T)]);
    } catch (...) {
      for (size_t j = 0; j < i; ++j) {
        delete[] reinterpret_cast<char*>(array[j]);
      }
      throw;
    }
  }

  std::pair<size_t, size_t> current(start);
  for (size_t i = 0; i < number; ++i) {
    try {
      new(array[current.first] + current.second) T();
    } catch (...) {
      for (size_t j = 0; j < i; ++j) {
        current = change_position(current, -1);
        (array[current.first] + current.second)->~T();
      }
      for(size_t j = 0; j < array.size(); ++j) {
        delete[] reinterpret_cast<char*>(array[j]);
      }
      throw;
    }
    current = change_position(current, 1);
  }
  finish = current;
}

template <typename T>
Deque<T>::Deque(size_t number, const T& element): array(2 * (number / inner_size + 1)),
                                                  filled_size(number),
                                                  start(array.size() / 4 + 1, 0) {
  for (size_t i = 0; i < array.size(); ++i) {
    try {
      array[i] = reinterpret_cast<T*>(new char[inner_size * sizeof(T)]);
    } catch (...) {
      for (size_t j = 0; j < i; ++j) {
        delete[] reinterpret_cast<char*>(array[j]);
      }
      throw;
    }
  }

  std::pair<size_t, size_t> current(start);
  for (size_t i = 0; i < number; ++i) {
    try {
      new(array[current.first] + current.second) T(element);
    } catch (...) {
      for (size_t j = 0; j < i; ++j) {
        current = change_position(current, -1);
        (array[current.first] + current.second)->~T();
      }
      for(size_t j = 0; j < array.size(); ++j) {
        delete[] reinterpret_cast<char*>(array[j]);
      }
      throw;
    }
    current = change_position(current, 1);
  }
  finish = current;
}

template <typename T>
Deque<T>::~Deque() {
  std::pair<size_t, size_t> current(start);
  for (size_t i = 0; i < filled_size; ++i) {
    (array[current.first] + current.second)->~T();
    current = change_position(current, 1);
  }
  for(size_t j = 0; j < array.size(); ++j) {
    delete[] reinterpret_cast<char*>(array[j]);
  }
}

template <typename T>
Deque<T>& Deque<T>::operator=(const Deque& other) {
  Deque copy(other);
  swap(copy);
  return *this;
}

template <typename T>
T& Deque<T>::operator[](size_t index) {
  std::pair<size_t, size_t> current = change_position(start, index);
  return array[current.first][current.second];
}

template <typename T>
const T& Deque<T>::operator[](size_t index) const {
  std::pair<size_t, size_t> current = change_position(start, index);
  return array[current.first][current.second];
}

template <typename T>
T& Deque<T>::at(size_t index) {
  if (index >= filled_size) {
    throw std::out_of_range("function at()");
  }
  std::pair<size_t, size_t> current = change_position(start, index);
  return array[current.first][current.second];
}

template <typename T>
const T& Deque<T>::at(size_t index) const {
  if (index >= filled_size) {
    throw std::out_of_range("function at()");
  }
  std::pair<size_t, size_t> current = change_position(start, index);
  return array[current.first][current.second];
}

template <typename T>
void Deque<T>::push_back(const T& element) {
  while (change_position(finish, 1).first >= array.size()) {
    increase(array.size() * 2);
  }
  new(array[finish.first] + finish.second) T(element);
  finish = change_position(finish, 1);
  ++filled_size;
}

template <typename T>
void Deque<T>::push_front(const T& element) {
  while (start.first == 0 and start.second == 0) {
    increase(array.size() * 2);
  }
  start = change_position(start, -1);
  new(array[start.first] + start.second) T(element);
  ++filled_size;
}

template <typename T>
void Deque<T>::pop_back() {
  finish = change_position(finish, -1);
  (array[finish.first] + finish.second)->~T();
  --filled_size;
}

template <typename T>
void Deque<T>::pop_front() {
  (array[start.first] + start.second)->~T();
  start = change_position(start, 1);
  --filled_size;
}

template <typename T>
void Deque<T>::insert(iterator iter, T element) {
  iterator current(iter);
  T current_data = *current;
  for (; current < end(); ++current) {
    current_data = *current;
    *current = element;
    element = current_data;
  }

  new(array[finish.first] + finish.second) T(element);
  finish = change_position(finish, 1);
  ++filled_size;
  if (finish.first >= array.size()) {
    increase(array.size() * 2);
  }
}

template <typename T>
void Deque<T>::erase(iterator iter) {
  iterator current(iter);
  iterator next = current + 1;
  for (; next < end(); ++next) {
    *current = *next;
    ++current;
  }

  finish = change_position(finish, -1);
  (array[finish.first] + finish.second)->~T();
  --filled_size;
}

template <typename T>
template <bool is_const>
struct Deque<T>::general_iterator {
private:
  typename std::conditional<is_const, const T*, T*>::type inner_pointer;
  std::pair<size_t, size_t> position;
  typename std::conditional<is_const, T* const *, T**>::type out_pointer;

public:
  using difference_type = int;
  using value_type = T;
  using reference = typename std::conditional<is_const, const T&, T&>::type;
  using pointer = typename std::conditional<is_const, const T*, T*>::type;
  using iterator_category = std::random_access_iterator_tag;

  general_iterator(std::pair<size_t, size_t> position,
                   typename std::conditional<is_const, T* const *, T**>::type out_pointer):
                  inner_pointer(*out_pointer + position.second),
                  position(position), out_pointer(out_pointer) {}

  typename std::conditional<is_const, const T&, T&>::type operator*() const {
    return *inner_pointer;
  }

  typename std::conditional<is_const, const T*, T*>::type operator->() const {
    return inner_pointer;
  }

  general_iterator& operator++() {
    size_t before = position.first;
    position = change_position(position, 1);
    out_pointer += position.first - before;
    inner_pointer = *out_pointer + position.second;
    return *this;
  }

  general_iterator operator++(int) {
    general_iterator new_iter(*this);
    ++(*this);
    return new_iter;
  }

  general_iterator& operator--() {
    size_t before = position.first;
    position = change_position(position, -1);
    out_pointer -= before - position.first;
    inner_pointer = *out_pointer + position.second;
    return *this;
  }

  general_iterator operator--(int) {
    general_iterator new_iter(*this);
    --(*this);
    return new_iter;
  }

  general_iterator& operator+=(int number) {
    int before = position.first;
    position = change_position(position, number);
    out_pointer += static_cast<int>(position.first) - before;
    inner_pointer = *out_pointer + position.second;
    return *this;
  }

  general_iterator& operator-=(int number) {
    int before = position.first;
    position = change_position(position, -number);
    out_pointer += static_cast<int>(position.first) - before;
    inner_pointer = *out_pointer + position.second;
    return *this;
  }

  general_iterator operator+(int number) const {
    general_iterator new_iter(*this);
    new_iter += number;
    return new_iter;
  }

  general_iterator operator-(int number) const {
    general_iterator new_iter(*this);
    new_iter -= number;
    return new_iter;
  }

  int operator-(const_iterator other) const {
    std::pair<int, int> this_pos = position;
    std::pair<int, int> other_pos = other.Position();

    if (this_pos.first == other_pos.first) {
      return this_pos.second - other_pos.second;
    }
    bool negative = false;
    if (this_pos.first < other_pos.first) {
      std::swap(this_pos, other_pos);
      negative = true;
    }

    int difference = (this_pos.first - other_pos.first - 1) * 16;
    difference += 16 - other_pos.second - 1;
    difference += this_pos.second + 1;
    return (negative) ? -difference : difference;
  }

  bool operator==(const_iterator other) const {
    return (*this - other) == 0;
  }

  bool operator!=(const_iterator other) const {
    return (*this - other) != 0;
  }

  bool operator>(const_iterator other) const {
    return (*this - other) > 0;
  }

  bool operator<(const_iterator other) const {
    return other > *this;
  }

  bool operator<=(const_iterator other) const {
    return !(other < *this);
  }

  bool operator>=(const_iterator other) const {
    return !(other > *this);
  }

  operator const_iterator() const {
    return const_iterator(position, out_pointer);
  }

  std::pair<size_t, size_t> Position() {
    return position;
  }
};