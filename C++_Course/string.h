#include <algorithm>
#include <cstring>
#include <iostream>

class String {
 private:
  char* str;
  size_t str_size;
  size_t str_capacity;
  void swap_(String& other);
  void resize_capacity(size_t new_capacity);
  bool substring_from_here(const String& substring, size_t index) const;

 public:
  String()
      : str(new char[1]),
        str_size(0),
        str_capacity(1) {
    str[0] = '\0';
  }
  String(const char* string)
      : str(new char[2 * std::strlen(string)]),
        str_size(std::strlen(string)),
        str_capacity(2 * std::strlen(string)) {
    std::memcpy(str, string, str_size);
  }
  String(size_t n, char c);
  String(char c)
      : String(1, c){};
  String(const String& other)
      : str(new char[other.str_capacity]),
        str_size(other.str_size),
        str_capacity(other.str_capacity) {
    std::memcpy(str, other.str, other.str_size);
  }

  ~String() {
    delete[] str;
  }

  String& operator=(const String& other);
  char& operator[](size_t index) {
    return str[index];
  }
  const char& operator[](size_t index) const {
    return str[index];
  }
  size_t length() const {
    return str_size;
  }
  size_t size() const {
    return str_size;
  }
  size_t capacity() const {
    return str_capacity - 1;
  }
  void push_back(char c);
  void pop_back();
  char& front() {
    return str[0];
  }
  const char& front() const {
    return str[0];
  }
  char& back() {
    return str[str_size - 1];
  }
  const char& back() const {
    return str[str_size - 1];
  }
  String& operator+=(const String& other);
  String substr(size_t start, size_t count) const;
  size_t find(const String& substring) const;
  size_t rfind(const String& substring) const;
  bool empty() const {
    return str_size == 0;
  }
  void clear();
  void shrink_to_fit();
  char* data() {
    return str;
  }
  const char* data() const {
    return str;
  }
};

void String::swap_(String& other) {
  std::swap(str, other.str);
  std::swap(str_size, other.str_size);
  std::swap(str_capacity, other.str_capacity);
}

void String::resize_capacity(size_t new_capacity) {
  str_capacity = new_capacity;
  char* new_str = new char[str_capacity];
  std::memcpy(new_str, str, str_size);
  delete[] str;
  str = new_str;
}

String::String(size_t n, char c)
    : str(new char[2 * n]),
      str_size(n),
      str_capacity(2 * n) {
  std::fill(str, str + n, c);
  str[n] = '\0';
}

bool operator==(const String& first, const String& second) {
  if (first.size() != second.size()) {
    return false;
  }
  for (size_t i = 0; i < first.size(); ++i) {
    if (first[i] != second[i]) {
      return false;
    }
  }
  return true;
}

String& String::operator=(const String& other) {
  if (*this == other) {
    return *this;
  }
  String copy(other);
  swap_(copy);
  return *this;
}

bool operator<(const String& first, const String& second) {
  for (size_t i = 0; i < std::min(first.size(), second.size()); ++i) {
    if (first[i] < second[i]) {
      return true;
    } else if (first[i] > second[i]) {
      return false;
    }
  }
  return first.size() < second.size();
}

bool operator!=(const String& first, const String& second) {
  return not(first == second);
}
bool operator>(const String& first, const String& second) {
  return second < first;
}
bool operator<=(const String& first, const String& second) {
  return not(second < first);
}
bool operator>=(const String& first, const String& second) {
  return not(first < second);
}

void String::push_back(char c) {
  if (str_size == str_capacity - 1) {
    resize_capacity(2 * (str_capacity + 1));
  }
  str[str_size] = c;
  ++str_size;
  str[str_size] = '\0';
}

void String::pop_back() {
  --str_size;
  str[str_size] = '\0';
}

String& String::operator+=(const String& other) {
  if (str_size + other.str_size >= str_capacity - 1) {
    resize_capacity(2 * (str_size + other.str_size));
  }
  std::memcpy(&str[str_size], other.str, other.str_size + 1);
  str_size += other.str_size;
  return *this;
}

String operator+(const String& first, const String& second) {
  String result(first);
  result += second;
  return result;
}

String String::substr(size_t start, size_t count) const {
  String new_str(count, '0');
  for (size_t i = 0; i < count; ++i) {
    new_str[i] = str[start + i];
  }
  return new_str;
}

bool String::substring_from_here(const String& substring, size_t index) const {
  size_t i = 0;
  for (; i < substring.str_size; ++i) {
    if (substring.str[i] != str[index + i]) {
      return false;
    }
  }
  return true;
}

size_t String::find(const String& substring) const {
  size_t start = 0;
  for (; start <= str_size - substring.str_size; ++start) {
    if (substring_from_here(substring, start)) {
      return start;
    }
  }
  return str_size;
}

size_t String::rfind(const String& substring) const {
  size_t finish = str_size - 1;
  for (; finish >= substring.str_size - 1; --finish) {
    if (substring_from_here(substring, finish - substring.str_size + 1)) {
      return finish - substring.str_size + 1;
    }
  }
  return str_size;
}

void String::clear() {
  str_size = 0;
  str[str_size] = '\0';
}

void String::shrink_to_fit() {
  if (str_size == str_capacity - 1) {
    return;
  }
  resize_capacity(str_size + 1);
}

std::ostream& operator<<(std::ostream& out, const String& string) {
  out << string.data();
  return out;
}

std::istream& operator>>(std::istream& in, String& string) {
  string.clear();
  char c;
  if (in.peek() == '\n') {
    in.get();
  }
  while (in.peek() != '\n' and in >> c and not std::isspace(c)) {
    string.push_back(c);
  }
  return in;
}