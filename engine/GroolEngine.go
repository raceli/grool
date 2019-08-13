package engine

import (
	"github.com/juju/errors"
	"github.com/newm4n/grool/context"
	"github.com/newm4n/grool/model"
	log "github.com/sirupsen/logrus"
	"sort"
)

func NewGroolEngine() *Grool {
	return &Grool{
		MaxCycle: 5000,
	}
}

type Grool struct {
	MaxCycle uint64
}

func (g *Grool) Execute(dataCtx *context.DataContext, knowledge *model.KnowledgeBase) error {
	defunc := &context.GroolFunctions{}
	kctx := &context.KnowledgeContext{}
	rctx := &context.RuleContext{}
	dataCtx.Add("DEFUNC", defunc)

	for _, v := range knowledge.RuleEntries {
		v.Initialize(kctx, rctx, dataCtx)
	}

	var cycle uint64

	/*
		Un-limitted loop as long as there are rule to execute.
		We need to add safety mechanism to detect unlimitted loop as there are posibility executed rule are not changing
		data context which makes rules to get executed again and again.
	*/
	for true {
		cycle++

		if cycle > g.MaxCycle {
			return errors.Errorf("Grool successfully selected rule candidate for execution after %d cycles, this could possibly caused by rule entry(s) that keep added into execution pool but when executed it does not change any data in context. Please evaluate your rule entries \"When\" and \"Then\" scope. You can adjust the maximum cycle using Grool.MaxCycle variable.", g.MaxCycle)
		}

		// Select all rule entry that can be executed.
		runnable := make([]*model.RuleEntry, 0)
		for _, v := range knowledge.RuleEntries {
			// test if this rule entry v can execute.
			can, err := v.CanExecute()
			if err != nil {
				log.Errorf("Failed testing condition for rule : %s. Got error %v", v.RuleName, err)
				return errors.Trace(err)
			}
			// if can, add into runnable array
			if can {
				runnable = append(runnable, v)
			}
		}

		// If there are rules to execute, sort them by their Salience
		if len(runnable) > 0 {
			if len(runnable) > 1 {
				sort.SliceStable(runnable, func(i, j int) bool {
					return runnable[i].Salience > runnable[j].Salience
				})
			}
			// Start rule execution cycle.
			// We assume that none of the runnable rule will change variable so we set it to true.
			cycleDone := true

			for _, r := range runnable {
				// reset the counter to 0 to detect if there are variable change.
				dataCtx.VariableChangeCount = 0
				//log.Infof("Executing rule : %s. Salience %d", r.RuleName, r.Salience)
				err := r.Execute()
				if err != nil {
					log.Errorf("Failed execution rule : %s. Got error %v", r.RuleName, err)
					return errors.Trace(err)
				}
				//if there is a variable change, restart the cycle.
				if dataCtx.VariableChangeCount > 0 {
					cycleDone = false
					break
				}
				// this point means no variable change, so we move to the next rule entry.
			}
			// if cycleDone is true, we are done.
			if cycleDone {
				break
			}
		} else {
			// No more rule can be executed, so we are done here.
			break
		}
	}
	log.Infof("Finished Rules execution. Total #%d cycles.", cycle)
	return nil
}
