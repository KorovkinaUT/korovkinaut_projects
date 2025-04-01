from random import choice


class Player:
    def __init__(self, arrows_number):
        self.location = choice(range(1, 21))
        self.arrows = arrows_number

    def move(self, new_position):
        self.location = new_position

    def shoot(self, cave, arrow_way):
        room_number = 0
        self.arrows -= 1
        arrow_location = self.location
        for room in arrow_way:
            room_number += 1
            if room_number > 5 or room not in cave.adjacent_rooms(arrow_location):
                arrow_location = choice(range(1, 21))
                print("You missed.")
                return arrow_location
            arrow_location = room
            if arrow_location == self.location:
                return arrow_location
            if arrow_location in cave.wumpuses:
                return arrow_location
        return arrow_location
