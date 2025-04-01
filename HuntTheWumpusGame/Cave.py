from random import choice


class Cave:
    tunnels = [[1, 2], [1, 3], [1, 8], [2, 1], [2, 4], [2, 10], [3, 1], [3, 5], [3, 6], [4, 2], [4, 5], [4, 12], [5, 4],
               [5, 3], [5, 14], [6, 3], [6, 7], [6, 15], [7, 6], [7, 8], [7, 16], [8, 1], [8, 7], [8, 9], [9, 8],
               [9, 10], [9, 17], [10, 9], [10, 2], [10, 11], [11, 10], [11, 12], [11, 18], [12, 11], [12, 4], [12, 13],
               [13, 12], [13, 19], [13, 14], [14, 13], [14, 5], [14, 15], [15, 14], [15, 6], [15, 20], [16, 7],
               [16, 17], [16, 20], [17, 16], [17, 9], [17, 18], [18, 17], [18, 11], [18, 19], [19, 18], [19, 13],
               [19, 20], [20, 19], [20, 15], [20, 16]]
    rooms = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]

    def __init__(self):
        self.tunnels = Cave.tunnels.copy()
        self.rooms = Cave.rooms.copy()
        for i in range(3):
            first = choice(self.rooms)
            self.rooms.remove(first)
            second = choice(self.rooms)
            self.swap_rooms(first, second)
            self.rooms.remove(second)

        self.rooms = Cave.rooms.copy()
        self.bats = set()
        self.wumpuses = set()
        self.holes = set()

    def swap_rooms(self, first, second):
        for tunnel in self.tunnels:
            if tunnel[0] == first:
                tunnel[0] = second
            elif tunnel[0] == second:
                tunnel[0] = first
            if tunnel[1] == first:
                tunnel[1] = second
            elif tunnel[1] == second:
                tunnel[1] = first

    def position_enemies(self, player_position, wumpuses_number, bats_number, holes_number=2):
        self.rooms.remove(player_position)
        for i in range(wumpuses_number):
            location = choice(self.rooms)
            self.wumpuses.add(location)
        for i in range(bats_number):
            location = choice(self.rooms)
            self.rooms.remove(location)
            self.bats.add(location)
        for i in range(holes_number):
            location = choice(self.rooms)
            self.rooms.remove(location)
            self.holes.add(location)
        self.rooms = Cave.rooms.copy()

    def adjacent_rooms(self, room):
        adjacent = set()
        for tunnel in self.tunnels:
            if tunnel[0] == room:
                adjacent.add(tunnel[1])
        return adjacent

    def move_wumpuses(self):
        new_wumpuses = set()
        for wumpus in self.wumpuses:
            variants = self.adjacent_rooms(wumpus)
            variants.add(wumpus)
            new_wumpuses.add(choice(list(variants)))
        self.wumpuses = new_wumpuses.copy()
