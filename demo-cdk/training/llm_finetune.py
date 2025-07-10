#!/usr/bin/env python
"""Fine-tune an open-source LLM on architecture Q&A pairs.

This script is designed to run inside the official HuggingFace SageMaker DLC
image (PyTorch / Transformers).  It expects the following environment
variables provided by SageMaker training jobs:

  * SM_CHANNEL_TRAIN – directory with one or more .jsonl files where each
    line contains {"prompt": ..., "completion": ...}
  * SM_MODEL_DIR     – where to save the final model/artifacts
  * SM_NUM_GPUS      – number of available GPUs (string)

The hyper-parameters are passed as command-line args from the SageMaker
training job definition.  We support the most common ones (epochs, lr, batch)
with sensible defaults.
"""

import argparse
import json
import os
from pathlib import Path

import torch
from datasets import load_dataset, Dataset
from transformers import AutoTokenizer, AutoModelForCausalLM, TrainingArguments, Trainer
from peft import LoraConfig, get_peft_model


HF_CACHE = os.environ.get("HF_HOME", "/opt/ml/processing/hf-cache")


def load_training_data(data_dir: Path) -> Dataset:
    """Loads jsonl files under *data_dir* into a 🤗 Dataset"""
    files = list(data_dir.glob("*.jsonl"))
    if not files:
        raise FileNotFoundError(f"No .jsonl files found in {data_dir}")

    rows = []
    for fp in files:
        with open(fp) as f:
            for line in f:
                obj = json.loads(line)
                prompt = obj.get("prompt")
                completion = obj.get("completion")
                if prompt and completion:
                    rows.append({"text": f"{prompt}\n{completion}"})
    return Dataset.from_list(rows)


def tokenize_dataset(dataset: Dataset, tokenizer, block_size: int = 1024):
    def _tokenize(example):
        return tokenizer(example["text"], truncation=True, max_length=block_size)

    return dataset.map(_tokenize, batched=True, remove_columns=["text"])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model_name_or_path", default="tiiuae/falcon-7b-instruct")
    parser.add_argument("--num_train_epochs", type=int, default=1)
    parser.add_argument("--learning_rate", type=float, default=2e-5)
    parser.add_argument("--per_device_train_batch_size", type=int, default=2)
    parser.add_argument("--logging_steps", type=int, default=10)
    parser.add_argument("--max_seq_length", type=int, default=1024)
    args = parser.parse_args()

    train_dir = Path(os.environ["SM_CHANNEL_TRAIN"])
    model_dir = Path(os.environ["SM_MODEL_DIR"])

    tokenizer = AutoTokenizer.from_pretrained(args.model_name_or_path, cache_dir=HF_CACHE)
    tokenizer.pad_token = tokenizer.eos_token

    base_model = AutoModelForCausalLM.from_pretrained(
        args.model_name_or_path,
        cache_dir=HF_CACHE,
        torch_dtype=torch.float16 if torch.cuda.is_available() else torch.float32,
    )

    # LoRA config (lightweight fine-tune)
    lora_cfg = LoraConfig(
        r=8,
        lora_alpha=32,
        target_modules=["q_proj", "v_proj"],
        lora_dropout=0.05,
        bias="none",
        task_type="CAUSAL_LM",
    )

    model = get_peft_model(base_model, lora_cfg)

    raw_ds = load_training_data(train_dir)
    tokenized_ds = tokenize_dataset(raw_ds, tokenizer, args.max_seq_length)

    training_args = TrainingArguments(
        output_dir=str(model_dir),
        per_device_train_batch_size=args.per_device_train_batch_size,
        learning_rate=args.learning_rate,
        num_train_epochs=args.num_train_epochs,
        logging_steps=args.logging_steps,
        save_total_limit=1,
        fp16=torch.cuda.is_available(),
        optim="adamw_torch",
        report_to=["tensorboard"],
    )

    trainer = Trainer(
        model=model,
        args=training_args,
        train_dataset=tokenized_ds,
        tokenizer=tokenizer,
    )

    trainer.train()

    # Save adapter config + base model ref
    trainer.save_model()
    tokenizer.save_pretrained(model_dir)

    # Write success marker for SageMaker
    with open(model_dir / "complete", "w") as f:
        f.write("success")


if __name__ == "__main__":
    main()