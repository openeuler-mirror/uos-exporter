package main

import (
	"fmt"
	"github.com/tidwall/gjson"
)

func main() {
	jsonData := `{
    "nftables": [
        {
            "metainfo": {
                "version": "1.0.9",
                "release_name": "Old Doc Yak #3",
                "json_schema_version": 1
            }
        },
        {
            "table": {
                "family": "inet",
                "name": "lxc",
                "handle": 11
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "lxc",
                "name": "input",
                "handle": 1,
                "type": "filter",
                "hook": "input",
                "prio": 0,
                "policy": "accept"
            }
        },
        {
            "chain": {
                "family": "inet",
                "table": "lxc",
                "name": "forward",
                "handle": 6,
                "type": "filter",
                "hook": "forward",
                "prio": 0,
                "policy": "accept"
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "input",
                "handle": 3,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "udp",
                                    "field": "dport"
                                }
                            },
                            "right": {
                                "set": [
                                    53,
                                    67
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "input",
                "handle": 5,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "tcp",
                                    "field": "dport"
                                }
                            },
                            "right": {
                                "set": [
                                    53,
                                    67
                                ]
                            }
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "forward",
                "handle": 7,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "iifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "rule": {
                "family": "inet",
                "table": "lxc",
                "chain": "forward",
                "handle": 8,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "meta": {
                                    "key": "oifname"
                                }
                            },
                            "right": "lxcbr0"
                        }
                    },
                    {
                        "accept": null
                    }
                ]
            }
        },
        {
            "table": {
                "family": "ip",
                "name": "lxc",
                "handle": 12
            }
        },
        {
            "chain": {
                "family": "ip",
                "table": "lxc",
                "name": "postrouting",
                "handle": 1,
                "type": "nat",
                "hook": "postrouting",
                "prio": 100,
                "policy": "accept"
            }
        },
        {
            "rule": {
                "family": "ip",
                "table": "lxc",
                "chain": "postrouting",
                "handle": 2,
                "expr": [
                    {
                        "match": {
                            "op": "==",
                            "left": {
                                "payload": {
                                    "protocol": "ip",
                                    "field": "saddr"
                                }
                            },
                            "right": {
                                "prefix": {
                                    "addr": "10.0.3.0",
                                    "len": 24
                                }
                            }
                        }
                    },
                    {
                        "match": {
                            "op": "!=",
                            "left": {
                                "payload": {
                                    "protocol": "ip",
                                    "field": "daddr"
                                }
                            },
                            "right": {
                                "prefix": {
                                    "addr": "10.0.3.0",
                                    "len": 24
                                }
                            }
                        }
                    },
                    {
                        "counter": {
                            "packets": 3,
                            "bytes": 335
                        }
                    },
                    {
                        "masquerade": null
                    }
                ]
            }
        }
    ]
}`
	if !gjson.Valid(jsonData) {
		fmt.Println("Invalid JSON data")
		return
	}
	if gjson.Get(jsonData, "nftables").Exists() {
		fmt.Println("Name field exists")
	} else {
		fmt.Println("Name field does not exist")
	}
	wd := gjson.Get(jsonData, "nftables")
	tables := wd.Get("#.table").Array()
	for _, table := range tables {
		fmt.Println("Table: ", table.Get("name").String())
	}
	//fmt.Printf("Tables: %#v\n", tables)
	chains := wd.Get("#.chain").Array()
	for _, chain := range chains {
		fmt.Println("Chain: ", chain.Get("name").String())
	}
	//fmt.Printf("Chains: %#v\n", chains)
	rules := wd.Get("#.rule").Array()
	for _, rule := range rules {
		fmt.Println("Rule: ", rule.Get("chain").String())
	}
	//fmt.Printf("Rules: %#v\n", rules)
}
