from Cave import Cave
from Player import Player
from InputGameSettings import input_game_settings
from Game import *
from time import time
from PrintStars import print_stars

print("Welcome to the game Hunt the Wumpus!\nTo see rules enter \"rules\". To find out the list of actions enter "
      "\"help\".\n>>>", end=" ")
action = input()
enemies = [1, 2]
while action != "quit":
    while action != "start" and action != "quit":
        if action == "configure":
            enemies = input_game_settings()
            if enemies[0] == 0:
                action = "quit"
                break
            print("Enter next action.\n>>>", end=" ")
        elif action == "rules":
            with open('Rules.txt', 'r') as file:
                file = file.read()
            print(file)
            print(">>>", end=" ")
        elif action == "help":
            with open('Help.txt', 'r') as file:
                file = file.read()
            print(file)
            print(">>>", end=" ")
        else:
            print("To see rules enter \"rules\". To find out the list of actions enter \"help\".\n>>>", end=" ")
        action = input()

    if action == "start":
        player = Player(enemies[0] + 4)
        cave = Cave()
        cave.position_enemies(player.location, enemies[0], enemies[1])
        start_time = time()
        action = game(cave, player)
        end_time = time()
        if action == "win":
            print_stars(enemies[0], end_time - start_time)
        elif action in ["restart", "quit"]:
            continue
        print(">>>", end=" ")
        action = input()
