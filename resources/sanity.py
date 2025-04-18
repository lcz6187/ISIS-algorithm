import sys
import re

def is_lowercase_string(s):
    pattern = r'^[a-z]+$'
    return bool(re.match(pattern, s))

def is_valid_positive_whole_number(s):
    pattern = r'^(0|[1-9]\d*)$'
    return bool(re.match(pattern, s))

def raise_assertError(line):
    raise AssertionError(f"Unexpected Line '{line.strip()}' found in output.")

def print_error_log(s):
    print("--------------------------------------Error----------------------------------------")
    print(s)
    print("--------------------------------------Error----------------------------------------")
    
def is_alphabetical_order(lst):
    return sorted(lst) == lst
                         
forbidden_substrings = ["Balances", "balances"]

if __name__=="__main__":
    for line in sys.stdin:
        if not line:
            print("End of input")
            break
        complete_line = line.strip()
        print(complete_line)
        
        #Check for correct 'BALANCES' prefix:
        for substring in forbidden_substrings:
            if complete_line.startswith(substring):
                print_error_log(f"line: '{line.strip()}'. 'BALANCES' should be in all caps")
                raise_assertError(line)
                
        if not complete_line.startswith("BALANCES"):
            print_error_log(f"line: '{line.strip()}'. Please comment out all debug logs, and/or check your output format for spelling errors in 'BALANCES'")
            raise_assertError(line)
        
        #Check for approriate account balances
        words = complete_line.split()
        if len(words) < 2:
            print_error_log(f"line: '{line.strip()}'. found in output. No Account balances printed")
            raise_assertError(line)
        
        accounts = []
        
        #Check for balance format
        for word in words[1:]:
            ac_balance = word.split(':')
            if len(ac_balance) != 2:
                print_error_log(f"line: '{line.strip()}'. Incorrect output format, missing account name, balance and/or ':' seperator")
                raise_assertError(line)
            if not is_lowercase_string(ac_balance[0]):
                print_error_log(f"line: '{line.strip()}'. Please check your account name output format")
                raise_assertError(line)
            if not is_valid_positive_whole_number(ac_balance[1]):
                print_error_log(f"line: '{line.strip()}'. Please check your account balance output format")
                raise_assertError(line)
            accounts.append(ac_balance[0])
            
        if not is_alphabetical_order(accounts):
            print_error_log(f"line: '{line.strip()}'. Accounts are not sorted alphabetically")
            raise_assertError(line)
        
        print("----------------------------------------ok-----------------------------------------")
